import {
  DataFrame,
  Field,
  formattedValueToString,
  getFieldDisplayName,
  getValueFormat,
  TimeRange,
  toURLRange,
  UrlQueryMap,
  UrlQueryValue,
  urlUtil,
} from '@grafana/data';
import { config } from '@grafana/runtime';
import { ComboboxOption } from '@grafana/ui';
import {
  DETAILS_DASHBOARD_PATH,
  LABEL_FIELDS,
  NO_VALUE,
  ROLE_ORDER,
  UNKNOWN_CLUSTER,
  URL_PARAMS,
} from './AuroraTreePanel.constants';
import { Messages } from './AuroraTreePanel.messages';
import {
  AuroraCluster,
  AuroraTreeOptions,
  ClusterFilters,
  FilteredClusters,
  InstanceRole,
  InstanceRow,
  SortValue,
} from './AuroraTreePanel.types';

type MetricKey = 'up' | 'readOnly' | 'qps' | 'storageBytes';

const findField = (frame: DataFrame, name: string, frames: DataFrame[]): Field | undefined => frame.fields
  .find((field) => field.name === name || getFieldDisplayName(field, frame, frames) === name);

const toNumber = (value: unknown): number | undefined => (
  typeof value === 'number' && Number.isFinite(value) ? value : undefined
);

const toLabel = (field: Field | undefined, index: number) => {
  const value = field?.values[index];

  return value === undefined || value === null ? '' : String(value);
};

const sumDefined = (total: number | undefined, value: number | undefined) => (
  value === undefined ? total : (total ?? 0) + value
);

const maxDefined = (max: number | undefined, value: number | undefined) => (
  value === undefined ? max : Math.max(max ?? value, value)
);

// Rows are joined on their labels, so the instant queries work with or without a merge transformation.
export const readInstances = (series: DataFrame[], options: AuroraTreeOptions): InstanceRow[] => {
  const rows = new Map<string, InstanceRow>();

  series.forEach((frame) => {
    const serviceField = findField(frame, LABEL_FIELDS.serviceName, series);

    if (!serviceField) {
      return;
    }

    const regionField = findField(frame, LABEL_FIELDS.region, series);
    const accountField = findField(frame, LABEL_FIELDS.awsAccountId, series);
    const clusterField = findField(frame, LABEL_FIELDS.clusterId, series);
    const metricFields: Array<[MetricKey, Field | undefined]> = [
      ['up', findField(frame, options.upField, series)],
      ['readOnly', findField(frame, options.readOnlyField, series)],
      ['qps', findField(frame, options.qpsField, series)],
      ['storageBytes', findField(frame, options.storageField, series)],
    ];

    serviceField.values.forEach((_, index) => {
      const serviceName = toLabel(serviceField, index);

      if (!serviceName) {
        return;
      }

      const region = toLabel(regionField, index);
      const awsAccountId = toLabel(accountField, index);
      const clusterId = toLabel(clusterField, index) || UNKNOWN_CLUSTER;
      const key = [region, awsAccountId, clusterId, serviceName].join('|');
      const row = rows.get(key) ?? {
        region,
        awsAccountId,
        clusterId,
        serviceName,
      };

      metricFields.forEach(([metric, field]) => {
        const value = toNumber(field?.values[index]);

        if (value !== undefined) {
          row[metric] = value;
        }
      });
      rows.set(key, row);
    });
  });

  return [...rows.values()];
};

export const getRole = (readOnly: number | undefined): InstanceRole => {
  if (readOnly === 0) {
    return 'writer';
  }

  if (readOnly === 1) {
    return 'reader';
  }

  return 'unknown';
};

// The unknown-cluster fallback has no aws_account_id, so it is ordered before the account.
const compareClusters = (a: AuroraCluster, b: AuroraCluster) => a.region.localeCompare(b.region)
  || Number(a.clusterId === UNKNOWN_CLUSTER) - Number(b.clusterId === UNKNOWN_CLUSTER)
  || a.awsAccountId.localeCompare(b.awsAccountId)
  || a.clusterId.localeCompare(b.clusterId);

export const groupClusters = (rows: InstanceRow[]): AuroraCluster[] => {
  const clusters = new Map<string, AuroraCluster>();

  rows.forEach((row) => {
    const key = [row.region, row.awsAccountId, row.clusterId].join('|');
    const cluster = clusters.get(key) ?? {
      key,
      region: row.region,
      awsAccountId: row.awsAccountId,
      clusterId: row.clusterId,
      instances: [],
      monitoredInstances: 0,
      exportersConnected: 0,
      writers: 0,
    };
    const role = getRole(row.readOnly);

    cluster.instances.push({
      serviceName: row.serviceName,
      role,
      up: row.up,
      qps: row.qps,
    });
    cluster.monitoredInstances += 1;
    cluster.exportersConnected += row.up ?? 0;
    cluster.writers += role === 'writer' ? 1 : 0;
    cluster.qps = sumDefined(cluster.qps, row.qps);
    // Every instance carries its cluster's storage, so summing would multiply it.
    cluster.storageBytes = maxDefined(cluster.storageBytes, row.storageBytes);
    clusters.set(key, cluster);
  });

  return [...clusters.values()]
    .map((cluster) => ({
      ...cluster,
      instances: [...cluster.instances].sort((a, b) => ROLE_ORDER.indexOf(a.role) - ROLE_ORDER.indexOf(b.role)
        || a.serviceName.localeCompare(b.serviceName)),
    }))
    .sort(compareClusters);
};

export const getWritersText = (writers: number) => (
  writers === 1 ? Messages.oneWriter : Messages.writersChanging
);

export const formatValue = (value: number | undefined, unit: string, decimals?: number) => (
  value === undefined ? NO_VALUE : formattedValueToString(getValueFormat(unit)(value, decimals))
);

export const getServiceUrl = (serviceName: string, timeRange: TimeRange) => {
  const { from, to } = toURLRange(timeRange.raw);

  return urlUtil.renderUrl(`${config.appSubUrl}${DETAILS_DASHBOARD_PATH}`, {
    'var-service_name': serviceName,
    from: String(from),
    to: String(to),
  });
};

const includesQuery = (value: string, query: string) => value.toLowerCase().includes(query);

// Plain substring match: subsequence matching would hit nearly every row, since all names share
// words like "database", "instance" and "aurora".
export const filterClusters = (clusters: AuroraCluster[], filters: ClusterFilters): FilteredClusters => {
  const query = filters.query.trim().toLowerCase();
  const autoExpanded = new Set<string>();
  const matching = clusters
    .filter((cluster) => !filters.accounts.length || filters.accounts.includes(cluster.awsAccountId))
    .filter((cluster) => !filters.clusters.length || filters.clusters.includes(cluster.clusterId))
    .reduce<AuroraCluster[]>((acc, cluster) => {
      const clusterFields = [cluster.clusterId, cluster.awsAccountId, cluster.region];

      if (!query || clusterFields.some((value) => includesQuery(value, query))) {
        acc.push(cluster);

        return acc;
      }

      const instances = cluster.instances.filter((instance) => includesQuery(instance.serviceName, query));

      if (instances.length) {
        autoExpanded.add(cluster.key);
        acc.push({ ...cluster, instances });
      }

      return acc;
    }, []);

  return { clusters: matching, autoExpanded };
};

const toOptions = (clusters: AuroraCluster[], getValue: (cluster: AuroraCluster) => string) => {
  const regions = new Map<string, Set<string>>();

  clusters.forEach((cluster) => {
    const value = getValue(cluster);

    if (value) {
      regions.set(value, (regions.get(value) ?? new Set()).add(cluster.region));
    }
  });

  return [...regions.entries()].map(([value, valueRegions]): ComboboxOption => ({
    label: value,
    value,
    description: [...valueRegions].sort().join(', '),
  }));
};

const byValue = (a: ComboboxOption, b: ComboboxOption) => Number(a.value === UNKNOWN_CLUSTER)
  - Number(b.value === UNKNOWN_CLUSTER) || a.value.localeCompare(b.value);

export const getAccountOptions = (clusters: AuroraCluster[]) => toOptions(
  clusters,
  (cluster) => cluster.awsAccountId,
).sort(byValue);

export const getClusterOptions = (clusters: AuroraCluster[], accounts: string[]) => toOptions(
  clusters.filter((cluster) => !accounts.length || accounts.includes(cluster.awsAccountId)),
  (cluster) => cluster.clusterId,
).sort(byValue);

const toList = (value: UrlQueryValue): string[] => {
  if (value === undefined || value === null) {
    return [];
  }

  return (Array.isArray(value) ? value : [value]).map(String).filter(Boolean);
};

export const readFilters = (search: UrlQueryMap): ClusterFilters => ({
  query: toList(search[URL_PARAMS.query])[0] ?? '',
  accounts: toList(search[URL_PARAMS.accounts]),
  clusters: toList(search[URL_PARAMS.clusters]),
});

// null removes a parameter from the URL.
export const toUrlQuery = ({ query, accounts, clusters }: ClusterFilters): UrlQueryMap => ({
  [URL_PARAMS.query]: query || null,
  [URL_PARAMS.accounts]: accounts.length ? accounts : null,
  [URL_PARAMS.clusters]: clusters.length ? clusters : null,
});

// Missing values sort last in both directions.
export const compareSortValues = (a: SortValue, b: SortValue, desc: boolean) => {
  if (a === b) {
    return 0;
  }

  if (a === undefined || b === undefined) {
    return a === undefined ? 1 : -1;
  }

  const result = typeof a === 'number' && typeof b === 'number' ? a - b : String(a).localeCompare(String(b));

  return desc ? -result : result;
};
