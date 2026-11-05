import {
  DataFrame,
  Field,
  formattedValueToString,
  getFieldDisplayName,
  getValueFormat,
  UrlQueryMap,
  UrlQueryValue,
  urlUtil,
} from '@grafana/data';
import { config } from '@grafana/runtime';
import { ComboboxOption } from '@grafana/ui';
import {
  DETAILS_DASHBOARD_PATH,
  ENGINES,
  LABEL_FIELDS,
  LEGACY_AURORA_ENGINE,
  METRIC_OPTIONS,
  MYSQL_SUMMARY_DASHBOARD_PATH,
  NODE_SUMMARY_DASHBOARD_PATH,
  NO_VALUE,
  POSTGRESQL_SUMMARY_DASHBOARD_PATH,
  ROLE_ORDER,
  SERVICE_TYPE_ENGINES,
  STORAGE_UNIT,
  THROUGHPUT_DECIMALS,
  THROUGHPUT_UNITS,
  UNKNOWN_GROUP,
  URL_PARAMS,
} from './AuroraTreePanel.constants';
import { Messages } from './AuroraTreePanel.messages';
import {
  AuroraTreeOptions,
  EngineFamily,
  EngineId,
  FilteredGroups,
  GroupFilters,
  GroupKind,
  GroupSelection,
  InstanceGroup,
  InstanceLabels,
  InstanceLink,
  InstanceMetric,
  InstanceRole,
  InstanceRow,
  RdsInstance,
  SortValue,
  Storage,
} from './AuroraTreePanel.types';

const LABEL_KEYS = Object.keys(LABEL_FIELDS) as Array<keyof InstanceLabels>;

const METRIC_KEYS = Object.keys(METRIC_OPTIONS) as InstanceMetric[];

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

const emptyRow = (): InstanceRow => ({
  serviceName: '',
  nodeName: '',
  nodeId: '',
  serviceType: '',
  region: '',
  awsAccountId: '',
  clusterId: '',
  awsEngine: '',
  instanceId: '',
  sourceInstanceId: '',
  sourceRegion: '',
});

// Labels keep their first non-empty value, metrics their first defined one.
const fillRow = (row: InstanceRow, from: InstanceRow): InstanceRow => {
  LABEL_KEYS.forEach((key) => {
    row[key] = row[key] || from[key];
  });
  METRIC_KEYS.forEach((key) => {
    row[key] = row[key] ?? from[key];
  });

  return row;
};

// A frame with service_name holds service metrics, one with only node_id node metrics (the AWS labels
// of aws_rds_instance_info, storage). Rows are joined on those labels, so the queries work with or
// without a merge transformation.
export const readInstances = (series: DataFrame[], options: AuroraTreeOptions): InstanceRow[] => {
  const services = new Map<string, InstanceRow>();
  const nodes = new Map<string, InstanceRow>();

  series.forEach((frame) => {
    const serviceField = findField(frame, LABEL_FIELDS.serviceName, series);
    const keyField = serviceField ?? findField(frame, LABEL_FIELDS.nodeId, series);
    const rows = serviceField ? services : nodes;

    if (!keyField) {
      return;
    }

    const labelFields = LABEL_KEYS.map((key) => [key, findField(frame, LABEL_FIELDS[key], series)] as const);
    const metricFields = METRIC_KEYS
      .map((key) => [key, findField(frame, options[METRIC_OPTIONS[key]], series)] as const);

    keyField.values.forEach((_, index) => {
      const key = toLabel(keyField, index);

      if (!key) {
        return;
      }

      const row = rows.get(key) ?? emptyRow();

      labelFields.forEach(([label, field]) => {
        row[label] = row[label] || toLabel(field, index);
      });
      metricFields.forEach(([metric, field]) => {
        const value = toNumber(field?.values[index]);

        if (value !== undefined) {
          row[metric] = value;
        }
      });
      rows.set(key, row);
    });
  });

  return [...services.values()].map((service) => {
    const node = nodes.get(service.nodeId);

    return node ? fillRow(service, node) : service;
  });
};

const isEngineId = (value: string): value is EngineId => Object.keys(ENGINES).includes(value);

// Without AWS details the engine falls back to the plain engine of the PMM service type.
export const getEngine = ({ awsEngine, serviceType }: InstanceRow): EngineId | undefined => {
  const engine = awsEngine === LEGACY_AURORA_ENGINE ? 'aurora-mysql' : awsEngine;

  return isEngineId(engine) ? engine : SERVICE_TYPE_ENGINES[serviceType];
};

const getFamily = (engine: EngineId | undefined): EngineFamily | undefined => (
  engine && ENGINES[engine].family
);

const toRole = (readOnly: number | undefined): InstanceRole => {
  if (readOnly === 0) {
    return 'writer';
  }

  return readOnly === 1 ? 'reader' : 'unknown';
};

// Aurora MySQL readers have innodb_read_only ON, RDS MySQL and MariaDB replicas read_only ON (with
// innodb_read_only OFF), and PostgreSQL readers are in recovery.
export const getRole = (row: InstanceRow, engine: EngineId | undefined): InstanceRole => {
  if (getFamily(engine) === 'postgresql') {
    return toRole(row.inRecovery);
  }

  if (engine === 'aurora-mysql') {
    return toRole(row.innodbReadOnly);
  }

  if (row.awsEngine) {
    return toRole(row.readOnly);
  }

  // Without AWS details Aurora cannot be told apart, and each engine marks its readers in one of the two.
  if (row.innodbReadOnly === 1 || row.readOnly === 1) {
    return 'reader';
  }

  return row.innodbReadOnly === 0 && row.readOnly === 0 ? 'writer' : 'unknown';
};

interface InstanceRef {
  region: string;
  awsAccountId: string;
  instanceId: string;
}

const refKey = ({ region, awsAccountId, instanceId }: InstanceRef) => [region, awsAccountId, instanceId].join('|');

const ownRef = (row: InstanceRow): InstanceRef => ({
  region: row.region,
  awsAccountId: row.awsAccountId,
  instanceId: row.instanceId || row.serviceName,
});

// aws_source_region is empty only for an unparseable source ARN; the source is then assumed to be local.
const sourceRef = (row: InstanceRow): InstanceRef => ({
  region: row.sourceRegion || row.region,
  awsAccountId: row.awsAccountId,
  instanceId: row.sourceInstanceId,
});

const isReplica = (row: InstanceRow) => Boolean(row.awsEngine && !row.clusterId && row.sourceInstanceId);

// Follows the replica sources up to the root, so a cascaded replica joins its primary's group.
const getPrimaryRef = (replica: InstanceRow, byRef: Map<string, InstanceRow>): InstanceRef => {
  const visited = new Set([replica]);
  let primary = sourceRef(replica);
  let parent = byRef.get(refKey(primary));

  while (parent && isReplica(parent) && !visited.has(parent)) {
    visited.add(parent);
    primary = sourceRef(parent);
    parent = byRef.get(refKey(primary));
  }

  return primary;
};

interface GroupId {
  key: string;
  kind: GroupKind;
  name: string;
  region: string;
  awsAccountId: string;
}

const toGroupId = (
  kind: GroupKind,
  { region, awsAccountId, instanceId }: InstanceRef,
  engine?: EngineId,
): GroupId => ({
  // Instances without AWS details are grouped per region and engine, so every group has one engine.
  key: [kind, region, awsAccountId, instanceId, kind === 'unknown' ? engine : ''].join('|'),
  kind,
  name: instanceId,
  region,
  awsAccountId,
});

const getGroupId = (
  row: InstanceRow,
  engine: EngineId | undefined,
  byRef: Map<string, InstanceRow>,
  primaries: Set<string>,
): GroupId => {
  if (row.clusterId) {
    return toGroupId('cluster', { ...ownRef(row), instanceId: row.clusterId });
  }

  if (!row.awsEngine) {
    return toGroupId('unknown', { region: row.region, awsAccountId: '', instanceId: UNKNOWN_GROUP }, engine);
  }

  if (row.sourceInstanceId) {
    return toGroupId('replication', getPrimaryRef(row, byRef));
  }

  return toGroupId(primaries.has(refKey(ownRef(row))) ? 'replication' : 'instance', ownRef(row));
};

const isAurora = (engine: EngineId | undefined) => Boolean(engine && ENGINES[engine].aurora);

const getInstanceStorage = (row: InstanceRow, engine: EngineId | undefined): Storage | undefined => (
  isAurora(engine) || row.freeStorageBytes === undefined ? undefined : { kind: 'free', bytes: row.freeStorageBytes }
);

const addStorage = (
  storage: Storage | undefined,
  row: InstanceRow,
  engine: EngineId | undefined,
): Storage | undefined => {
  if (isAurora(engine)) {
    // Every instance reports its cluster's volume, so summing would multiply it.
    const bytes = maxDefined(storage?.bytes, row.volumeUsedBytes);

    return bytes === undefined ? undefined : { kind: 'used', bytes };
  }

  if (row.freeStorageBytes === undefined) {
    return storage;
  }

  return storage
    ? { kind: 'free', bytes: Math.min(storage.bytes, row.freeStorageBytes), lowest: true }
    : { kind: 'free', bytes: row.freeStorageBytes };
};

// The group without AWS details has no aws_account_id, so it is ordered before the account.
const compareGroups = (a: InstanceGroup, b: InstanceGroup) => a.region.localeCompare(b.region)
  || Number(a.kind === 'unknown') - Number(b.kind === 'unknown')
  || a.awsAccountId.localeCompare(b.awsAccountId)
  || a.name.localeCompare(b.name)
  || a.key.localeCompare(b.key);

const compareInstances = (a: RdsInstance, b: RdsInstance) => (
  ROLE_ORDER.indexOf(a.role) - ROLE_ORDER.indexOf(b.role) || a.serviceName.localeCompare(b.serviceName)
);

// Aurora and Multi-AZ DB clusters group by cluster ID, a primary and its read replicas by the primary,
// any other instance stands alone.
export const groupInstances = (rows: InstanceRow[]): InstanceGroup[] => {
  const byRef = new Map(rows.filter((row) => row.awsEngine).map((row) => [refKey(ownRef(row)), row]));
  const primaries = new Set(rows.filter(isReplica).map((row) => refKey(getPrimaryRef(row, byRef))));
  const groups = new Map<string, InstanceGroup>();

  rows.forEach((row) => {
    const engine = getEngine(row);
    const role = getRole(row, engine);
    const { key, ...groupId } = getGroupId(row, engine, byRef, primaries);
    const group = groups.get(key) ?? {
      key,
      ...groupId,
      engine,
      instances: [],
      monitoredInstances: 0,
      exportersConnected: 0,
      writers: 0,
      unknownRoles: 0,
    };

    group.instances.push({
      serviceName: row.serviceName,
      nodeName: row.nodeName,
      instanceId: row.instanceId,
      region: row.region,
      engine,
      role,
      up: row.up,
      throughput: row.throughput,
      storage: getInstanceStorage(row, engine),
    });
    group.monitoredInstances += 1;
    group.exportersConnected += row.up ?? 0;
    group.writers += role === 'writer' ? 1 : 0;
    group.unknownRoles += role === 'unknown' ? 1 : 0;
    group.throughput = sumDefined(group.throughput, row.throughput);
    group.storage = addStorage(group.storage, row, engine);
    groups.set(key, group);
  });

  return [...groups.values()]
    .map((group) => ({ ...group, instances: [...group.instances].sort(compareInstances) }))
    .sort(compareGroups);
};

interface Writers {
  text: string;
  state: 'good' | 'warning' | 'unknown';
}

export const getWriters = ({ writers, unknownRoles }: InstanceGroup): Writers => {
  if (writers) {
    return { text: Messages.writers(writers), state: writers === 1 ? 'good' : 'warning' };
  }

  return unknownRoles
    ? { text: Messages.writerUnknown, state: 'unknown' }
    : { text: Messages.noWriter, state: 'warning' };
};

export const getEngineLabel = (engine: EngineId | undefined) => (engine ? ENGINES[engine].label : NO_VALUE);

export const formatValue = (value: number | undefined, unit: string, decimals?: number) => (
  value === undefined ? NO_VALUE : formattedValueToString(getValueFormat(unit)(value, decimals))
);

export const formatThroughput = (value: number | undefined, engine: EngineId | undefined) => {
  const family = getFamily(engine);

  return formatValue(value, family ? THROUGHPUT_UNITS[family] : 'none', THROUGHPUT_DECIMALS);
};

export const formatStorage = (storage: Storage | undefined) => {
  if (!storage) {
    return NO_VALUE;
  }

  const value = formatValue(storage.bytes, STORAGE_UNIT);

  if (storage.kind === 'used') {
    return Messages.storageUsed(value);
  }

  return storage.lowest ? Messages.storageLowestFree(value) : Messages.storageFree(value);
};

// No time range: the overview shows current state, so a dashboard opens with its own default range.
const getDashboardUrl = (path: string, params: UrlQueryMap) => urlUtil.renderUrl(
  `${config.appSubUrl}${path}`,
  params,
);

// The instance name opens the engine's main dashboard; the other links follow it.
export const getInstanceLinks = ({ serviceName, nodeName, engine }: RdsInstance) => {
  const serviceParams = { 'var-service_name': serviceName };
  // RDS-added instances have node_name equal to service_name, so that is the fallback.
  const node = nodeName || serviceName;
  const os: InstanceLink = {
    label: Messages.links.os,
    title: Messages.links.nodeSummary(node),
    url: getDashboardUrl(NODE_SUMMARY_DASHBOARD_PATH, { 'var-node_name': node }),
  };
  const mysqlSummary: InstanceLink = {
    label: Messages.links.mysql,
    title: Messages.links.mysqlSummary(serviceName),
    url: getDashboardUrl(MYSQL_SUMMARY_DASHBOARD_PATH, serviceParams),
  };

  if (engine === 'aurora-mysql') {
    return {
      service: {
        label: serviceName,
        title: Messages.links.details(serviceName),
        url: getDashboardUrl(DETAILS_DASHBOARD_PATH, serviceParams),
      },
      dashboards: [mysqlSummary, os],
    };
  }

  if (getFamily(engine) === 'postgresql') {
    return {
      service: {
        label: serviceName,
        title: Messages.links.postgresqlSummary(serviceName),
        url: getDashboardUrl(POSTGRESQL_SUMMARY_DASHBOARD_PATH, serviceParams),
      },
      dashboards: [os],
    };
  }

  return { service: { ...mysqlSummary, label: serviceName }, dashboards: [os] };
};

// The AWS instance ID and region, where the service name and the group do not show them already.
export const getInstanceDetails = (instance: RdsInstance, group: InstanceGroup) => [
  instance.instanceId && instance.instanceId !== instance.serviceName ? instance.instanceId : '',
  instance.region !== group.region ? instance.region : '',
].filter(Boolean).join(', ');

const includesQuery = (value: string, query: string) => value.toLowerCase().includes(query);

const isSelected = (selected: string[], value: string) => !selected.length || selected.includes(value);

const matchesSelection = (group: InstanceGroup, { regions, accounts, engines }: Omit<GroupSelection, 'groups'>) => (
  isSelected(regions, group.region)
  && isSelected(accounts, group.awsAccountId)
  && isSelected(engines, group.engine ?? '')
);

// Plain substring match: subsequence matching would hit nearly every row, since all names share
// words like "database", "instance" and "aurora".
export const filterGroups = (groups: InstanceGroup[], filters: GroupFilters): FilteredGroups => {
  const query = filters.query.trim().toLowerCase();
  const autoExpanded = new Set<string>();
  const matching = groups
    .filter((group) => matchesSelection(group, filters) && isSelected(filters.groups, group.name))
    .reduce<InstanceGroup[]>((acc, group) => {
      const groupFields = [group.name, group.awsAccountId, group.region, getEngineLabel(group.engine)];

      if (!query || groupFields.some((value) => includesQuery(value, query))) {
        acc.push(group);

        return acc;
      }

      const instances = group.instances.filter((instance) => includesQuery(instance.serviceName, query)
        || includesQuery(instance.instanceId, query));

      if (instances.length) {
        autoExpanded.add(group.key);
        acc.push({ ...group, instances });
      }

      return acc;
    }, []);

  return { groups: matching, autoExpanded };
};

// One option per distinct value; the description lists the regions the value occurs in.
const toOptions = (
  groups: InstanceGroup[],
  getValue: (group: InstanceGroup) => string,
  withRegions = true,
) => {
  const regions = new Map<string, Set<string>>();

  groups.forEach((group) => {
    const value = getValue(group);

    if (value) {
      regions.set(value, (regions.get(value) ?? new Set()).add(group.region));
    }
  });

  return [...regions.entries()]
    .map(([value, valueRegions]): ComboboxOption => ({
      label: value,
      value,
      ...(withRegions && { description: [...valueRegions].sort().join(', ') }),
    }))
    .sort((a, b) => Number(a.value === UNKNOWN_GROUP) - Number(b.value === UNKNOWN_GROUP)
      || a.value.localeCompare(b.value));
};

export const getRegionOptions = (groups: InstanceGroup[]) => toOptions(
  groups,
  (group) => group.region,
  false,
);

export const getAccountOptions = (groups: InstanceGroup[], regions: string[]) => toOptions(
  groups.filter((group) => isSelected(regions, group.region)),
  (group) => group.awsAccountId,
);

export const getEngineOptions = (groups: InstanceGroup[]): ComboboxOption[] => [
  ...new Set(groups.map((group) => group.engine).filter((engine): engine is EngineId => Boolean(engine))),
]
  .map((engine) => ({ label: ENGINES[engine].label, value: engine }))
  .sort((a, b) => a.label.localeCompare(b.label));

export const getGroupOptions = (groups: InstanceGroup[], selection: Omit<GroupSelection, 'groups'>) => toOptions(
  groups.filter((group) => matchesSelection(group, selection)),
  (group) => group.name,
);

const hasOption = (options: ComboboxOption[]) => (value: string) => options
  .some((option) => option.value === value);

// Drops selected accounts and groups that the selected regions, accounts and engines no longer offer.
export const narrowSelection = (groups: InstanceGroup[], selection: GroupSelection): GroupSelection => {
  const { regions, engines } = selection;
  const accounts = selection.accounts.filter(hasOption(getAccountOptions(groups, regions)));
  const groupOptions = getGroupOptions(groups, { regions, accounts, engines });

  return {
    regions,
    accounts,
    engines,
    groups: selection.groups.filter(hasOption(groupOptions)),
  };
};

const toList = (value: UrlQueryValue): string[] => {
  if (value === undefined || value === null) {
    return [];
  }

  return (Array.isArray(value) ? value : [value]).map(String).filter(Boolean);
};

export const readFilters = (search: UrlQueryMap): GroupFilters => ({
  query: toList(search[URL_PARAMS.query])[0] ?? '',
  regions: toList(search[URL_PARAMS.regions]),
  accounts: toList(search[URL_PARAMS.accounts]),
  engines: toList(search[URL_PARAMS.engines]),
  groups: toList(search[URL_PARAMS.groups]),
});

const toParam = (values: string[]) => (values.length ? values : null);

// null removes a parameter from the URL.
export const toUrlQuery = ({
  query, regions, accounts, engines, groups,
}: GroupFilters): UrlQueryMap => ({
  [URL_PARAMS.query]: query || null,
  [URL_PARAMS.regions]: toParam(regions),
  [URL_PARAMS.accounts]: toParam(accounts),
  [URL_PARAMS.engines]: toParam(engines),
  [URL_PARAMS.groups]: toParam(groups),
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
