import type { RdsInstanceRow } from 'types/rds.types';
import {
  DASHBOARD_URLS,
  ENGINES,
  LEGACY_AURORA_ENGINE,
  NO_VALUE,
  ROLE_ORDER,
  SERVICE_TYPE_ENGINES,
  THROUGHPUT_UNITS,
  UNKNOWN_GROUP,
  URL_PARAMS,
} from './AmazonRds.constants';
import { Messages } from './AmazonRds.messages';
import type {
  EngineFamily,
  EngineId,
  FilteredGroups,
  FilterOption,
  GroupColumnId,
  GroupFilters,
  GroupKind,
  GroupSelection,
  GroupSort,
  GroupType,
  InstanceGroup,
  InstanceLink,
  InstanceLinks,
  InstanceRole,
  RdsInstance,
  SortValue,
  Storage,
  WritersState,
} from './AmazonRds.types';

const sumDefined = (total: number | undefined, value: number | undefined) =>
  value === undefined ? total : (total ?? 0) + value;

const maxDefined = (max: number | undefined, value: number | undefined) =>
  value === undefined ? max : Math.max(max ?? value, value);

const isEngineId = (value: string): value is EngineId =>
  Object.keys(ENGINES).includes(value);

// Without AWS details the engine falls back to the plain engine of the PMM service type.
export const getEngine = ({
  awsEngine,
  serviceType,
}: RdsInstanceRow): EngineId | undefined => {
  const engine =
    awsEngine === LEGACY_AURORA_ENGINE ? 'aurora-mysql' : awsEngine;

  return isEngineId(engine) ? engine : SERVICE_TYPE_ENGINES[serviceType];
};

const getFamily = (engine: EngineId | undefined): EngineFamily | undefined =>
  engine && ENGINES[engine].family;

const isAurora = (engine: EngineId | undefined) =>
  !!engine && ENGINES[engine].aurora;

const toRole = (readOnly: number | undefined): InstanceRole => {
  if (readOnly === 0) {
    return 'writer';
  }

  return readOnly === 1 ? 'reader' : 'unknown';
};

// Aurora MySQL readers have innodb_read_only ON, RDS MySQL and MariaDB replicas read_only ON (with
// innodb_read_only OFF), and PostgreSQL readers are in recovery.
export const getRole = (
  row: RdsInstanceRow,
  engine: EngineId | undefined
): InstanceRole => {
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

const refKey = ({ region, awsAccountId, instanceId }: InstanceRef) =>
  [region, awsAccountId, instanceId].join('|');

const ownRef = (row: RdsInstanceRow): InstanceRef => ({
  region: row.region,
  awsAccountId: row.awsAccountId,
  instanceId: row.instanceId || row.serviceName,
});

// aws_source_region is empty only for an unparseable source ARN; the source is then assumed to be local.
const sourceRef = (row: RdsInstanceRow): InstanceRef => ({
  region: row.sourceRegion || row.region,
  awsAccountId: row.awsAccountId,
  instanceId: row.sourceInstanceId,
});

const isReplica = (row: RdsInstanceRow) =>
  !!(row.awsEngine && !row.clusterId && row.sourceInstanceId);

// Follows the replica sources up to the root, so a cascaded replica joins its primary's group.
const getPrimaryRef = (
  replica: RdsInstanceRow,
  byRef: Map<string, RdsInstanceRow>
): InstanceRef => {
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
  engine?: EngineId
): GroupId => ({
  // Instances without AWS details are grouped per region and engine, so every group has one engine.
  key: [
    kind,
    region,
    awsAccountId,
    instanceId,
    kind === 'unknown' ? engine : '',
  ].join('|'),
  kind,
  name: instanceId,
  region,
  awsAccountId,
});

const getGroupId = (
  row: RdsInstanceRow,
  engine: EngineId | undefined,
  byRef: Map<string, RdsInstanceRow>,
  primaries: Set<string>
): GroupId => {
  if (row.clusterId) {
    return toGroupId('cluster', { ...ownRef(row), instanceId: row.clusterId });
  }

  if (!row.awsEngine) {
    return toGroupId(
      'unknown',
      { region: row.region, awsAccountId: '', instanceId: UNKNOWN_GROUP },
      engine
    );
  }

  if (row.sourceInstanceId) {
    return toGroupId('replication', getPrimaryRef(row, byRef));
  }

  return toGroupId(
    primaries.has(refKey(ownRef(row))) ? 'replication' : 'instance',
    ownRef(row)
  );
};

const getInstanceStorage = (
  row: RdsInstanceRow,
  engine: EngineId | undefined
): Storage | undefined =>
  isAurora(engine) || row.freeStorageBytes === undefined
    ? undefined
    : { kind: 'free', bytes: row.freeStorageBytes };

const addStorage = (
  storage: Storage | undefined,
  row: RdsInstanceRow,
  engine: EngineId | undefined
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
    ? {
        kind: 'free',
        bytes: Math.min(storage.bytes, row.freeStorageBytes),
        lowest: true,
      }
    : { kind: 'free', bytes: row.freeStorageBytes };
};

// The group without AWS details has no aws_account_id, so it is ordered before the account.
const compareGroups = (a: InstanceGroup, b: InstanceGroup) =>
  a.region.localeCompare(b.region) ||
  Number(a.kind === 'unknown') - Number(b.kind === 'unknown') ||
  a.awsAccountId.localeCompare(b.awsAccountId) ||
  a.name.localeCompare(b.name) ||
  a.key.localeCompare(b.key);

const compareInstances = (a: RdsInstance, b: RdsInstance) =>
  ROLE_ORDER.indexOf(a.role) - ROLE_ORDER.indexOf(b.role) ||
  a.serviceName.localeCompare(b.serviceName);

// Aurora and Multi-AZ DB clusters group by cluster ID, a primary and its read replicas by the
// primary, any other instance stands alone.
export const groupInstances = (rows: RdsInstanceRow[]): InstanceGroup[] => {
  const byRef = new Map(
    rows.filter((row) => row.awsEngine).map((row) => [refKey(ownRef(row)), row])
  );
  const primaries = new Set(
    rows.filter(isReplica).map((row) => refKey(getPrimaryRef(row, byRef)))
  );
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
    .map((group) => ({
      ...group,
      instances: [...group.instances].sort(compareInstances),
    }))
    .sort(compareGroups);
};

export const getWriters = ({
  writers,
  unknownRoles,
}: InstanceGroup): { text: string; state: WritersState } => {
  if (writers) {
    return {
      text: Messages.writers(writers),
      state: writers === 1 ? 'good' : 'warning',
    };
  }

  return unknownRoles
    ? { text: Messages.writerUnknown, state: 'unknown' }
    : { text: Messages.noWriter, state: 'warning' };
};

export const getEngineLabel = (engine: EngineId | undefined) =>
  engine ? ENGINES[engine].label : NO_VALUE;

const BYTE_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'];

const BYTES_FORMAT = new Intl.NumberFormat('en-US', {
  maximumFractionDigits: 1,
});

const THROUGHPUT_FORMAT = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

export const formatBytes = (bytes: number) => {
  let value = bytes;
  let unit = 0;

  while (Math.abs(value) >= 1024 && unit < BYTE_UNITS.length - 1) {
    value /= 1024;
    unit += 1;
  }

  return `${BYTES_FORMAT.format(value)} ${BYTE_UNITS[unit]}`;
};

export const formatThroughput = (
  value: number | undefined,
  engine: EngineId | undefined
) => {
  if (value === undefined) {
    return NO_VALUE;
  }

  const family = getFamily(engine);
  const formatted = THROUGHPUT_FORMAT.format(value);

  return family ? `${formatted} ${THROUGHPUT_UNITS[family]}` : formatted;
};

export const formatStorage = (storage: Storage | undefined) => {
  if (!storage) {
    return NO_VALUE;
  }

  const value = formatBytes(storage.bytes);

  if (storage.kind === 'used') {
    return Messages.storageUsed(value);
  }

  return storage.lowest
    ? Messages.storageLowestFree(value)
    : Messages.storageFree(value);
};

// No time range: the page shows current state, so a dashboard opens with its own default range.
const getDashboardUrl = (path: string, variable: string, value: string) =>
  `${path}?${variable}=${encodeURIComponent(value)}`;

// The service name opens the engine's main dashboard; the other links follow it.
export const getInstanceLinks = ({
  serviceName,
  nodeName,
  engine,
}: RdsInstance): InstanceLinks => {
  // RDS-added instances have node_name equal to service_name, so that is the fallback.
  const node = nodeName || serviceName;
  const os: InstanceLink = {
    label: Messages.links.os,
    title: Messages.links.nodeSummary(node),
    url: getDashboardUrl(DASHBOARD_URLS.nodeSummary, 'var-node_name', node),
  };
  const mysqlSummary: InstanceLink = {
    label: Messages.links.mysql,
    title: Messages.links.mysqlSummary(serviceName),
    url: getDashboardUrl(
      DASHBOARD_URLS.mysqlSummary,
      'var-service_name',
      serviceName
    ),
  };

  if (engine === 'aurora-mysql') {
    return {
      service: {
        label: serviceName,
        title: Messages.links.details(serviceName),
        url: getDashboardUrl(
          DASHBOARD_URLS.auroraDetails,
          'var-service_name',
          serviceName
        ),
      },
      dashboards: [mysqlSummary, os],
    };
  }

  if (getFamily(engine) === 'postgresql') {
    return {
      service: {
        label: serviceName,
        title: Messages.links.postgresqlSummary(serviceName),
        url: getDashboardUrl(
          DASHBOARD_URLS.postgresqlSummary,
          'var-service_name',
          serviceName
        ),
      },
      dashboards: [os],
    };
  }

  return { service: { ...mysqlSummary, label: serviceName }, dashboards: [os] };
};

// The AWS instance ID and region, where the service name and the group do not show them already.
export const getInstanceDetails = (
  instance: RdsInstance,
  group: InstanceGroup
) =>
  [
    instance.instanceId && instance.instanceId !== instance.serviceName
      ? instance.instanceId
      : '',
    instance.region !== group.region ? instance.region : '',
  ]
    .filter(Boolean)
    .join(', ');

export const getGroupType = ({ kind, engine }: InstanceGroup): GroupType => {
  if (kind === 'cluster') {
    return isAurora(engine) ? 'aurora-cluster' : 'multi-az-cluster';
  }

  return kind === 'unknown' ? 'unavailable' : kind;
};

export const getGroupTypeLabel = (group: InstanceGroup) =>
  Messages.groupTypes[getGroupType(group)];

const includesQuery = (value: string, query: string) =>
  value.toLowerCase().includes(query);

const isSelected = (selected: string[], value: string) =>
  !selected.length || selected.includes(value);

const matchesSelection = (
  group: InstanceGroup,
  { regions, accounts, engines, types }: Omit<GroupSelection, 'groups'>
) =>
  isSelected(regions, group.region) &&
  isSelected(accounts, group.awsAccountId) &&
  isSelected(engines, group.engine ?? '') &&
  isSelected(types, getGroupType(group));

// Plain substring match: subsequence matching would hit nearly every row, since all names share
// words like "database", "instance" and "aurora".
export const filterGroups = (
  groups: InstanceGroup[],
  filters: GroupFilters
): FilteredGroups => {
  const query = filters.query.trim().toLowerCase();
  const autoExpanded = new Set<string>();
  const matching = groups
    .filter(
      (group) =>
        matchesSelection(group, filters) &&
        isSelected(filters.groups, group.name)
    )
    .reduce<InstanceGroup[]>((acc, group) => {
      const groupFields = [
        group.name,
        getGroupTypeLabel(group),
        group.awsAccountId,
        group.region,
        getEngineLabel(group.engine),
      ];

      if (!query || groupFields.some((value) => includesQuery(value, query))) {
        acc.push(group);

        return acc;
      }

      const instances = group.instances.filter(
        (instance) =>
          includesQuery(instance.serviceName, query) ||
          includesQuery(instance.instanceId, query)
      );

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
  withRegions = true
): FilterOption[] => {
  const regions = new Map<string, Set<string>>();

  groups.forEach((group) => {
    const value = getValue(group);

    if (value) {
      regions.set(value, (regions.get(value) ?? new Set()).add(group.region));
    }
  });

  return [...regions.entries()]
    .map(([value, valueRegions]) => ({
      label: value,
      value,
      ...(withRegions && {
        description: [...valueRegions].sort().join(', '),
      }),
    }))
    .sort(
      (a, b) =>
        Number(a.value === UNKNOWN_GROUP) - Number(b.value === UNKNOWN_GROUP) ||
        a.value.localeCompare(b.value)
    );
};

export const getRegionOptions = (groups: InstanceGroup[]) =>
  toOptions(groups, (group) => group.region, false);

export const getAccountOptions = (groups: InstanceGroup[], regions: string[]) =>
  toOptions(
    groups.filter((group) => isSelected(regions, group.region)),
    (group) => group.awsAccountId
  );

export const getEngineOptions = (groups: InstanceGroup[]): FilterOption[] =>
  [
    ...new Set(
      groups
        .map((group) => group.engine)
        .filter((engine): engine is EngineId => !!engine)
    ),
  ]
    .map((engine) => ({ label: ENGINES[engine].label, value: engine }))
    .sort((a, b) => a.label.localeCompare(b.label));

export const getTypeOptions = (groups: InstanceGroup[]): FilterOption[] =>
  [...new Set(groups.map(getGroupType))]
    .map((type) => ({ label: Messages.groupTypes[type], value: type }))
    .sort((a, b) => a.label.localeCompare(b.label));

export const getGroupOptions = (
  groups: InstanceGroup[],
  selection: Omit<GroupSelection, 'groups'>
) =>
  toOptions(
    groups.filter((group) => matchesSelection(group, selection)),
    (group) => group.name
  );

const hasOption = (options: FilterOption[]) => (value: string) =>
  options.some((option) => option.value === value);

// Drops selected accounts and groups that the selected regions, accounts and engines no longer offer.
export const narrowSelection = (
  groups: InstanceGroup[],
  selection: GroupSelection
): GroupSelection => {
  const { regions, engines, types } = selection;
  const accounts = selection.accounts.filter(
    hasOption(getAccountOptions(groups, regions))
  );
  const groupOptions = getGroupOptions(groups, {
    regions,
    accounts,
    engines,
    types,
  });

  return {
    regions,
    accounts,
    engines,
    types,
    groups: selection.groups.filter(hasOption(groupOptions)),
  };
};

export const readFilters = (params: URLSearchParams): GroupFilters => {
  const list = (name: string) => params.getAll(name).filter(Boolean);

  return {
    query: params.get(URL_PARAMS.query) ?? '',
    regions: list(URL_PARAMS.regions),
    accounts: list(URL_PARAMS.accounts),
    engines: list(URL_PARAMS.engines),
    types: list(URL_PARAMS.types),
    groups: list(URL_PARAMS.groups),
  };
};

// Keeps the other parameters of the URL; empty filters are left out.
export const writeFilters = (
  filters: GroupFilters,
  params: URLSearchParams
): URLSearchParams => {
  const next = new URLSearchParams(params);

  (Object.keys(URL_PARAMS) as Array<keyof GroupFilters>).forEach((key) => {
    const value = filters[key];

    next.delete(URL_PARAMS[key]);
    (Array.isArray(value) ? value : [value])
      .filter(Boolean)
      .forEach((item) => next.append(URL_PARAMS[key], item));
  });

  return next;
};

export const GROUP_SORT_VALUES: Record<
  GroupColumnId,
  (group: InstanceGroup) => SortValue
> = {
  region: (group) => group.region,
  awsAccountId: (group) => group.awsAccountId,
  name: (group) => group.name,
  type: (group) => getGroupTypeLabel(group),
  engine: (group) => group.engine && getEngineLabel(group.engine),
  monitoredInstances: (group) => group.monitoredInstances,
  exportersConnected: (group) => group.exportersConnected,
  writers: (group) => group.writers,
  throughput: (group) => group.throughput,
  storage: (group) => group.storage?.bytes,
};

// Missing values sort last in both directions.
export const compareSortValues = (
  a: SortValue,
  b: SortValue,
  desc: boolean
) => {
  if (a === b) {
    return 0;
  }

  if (a === undefined || b === undefined) {
    return a === undefined ? 1 : -1;
  }

  const result =
    typeof a === 'number' && typeof b === 'number'
      ? a - b
      : String(a).localeCompare(String(b));

  return desc ? -result : result;
};

const isColumnId = (id: string): id is GroupColumnId =>
  Object.keys(GROUP_SORT_VALUES).includes(id);

// Sorts by the first column of the table's sorting state; without one, groups keep their order.
export const sortGroups = (groups: InstanceGroup[], sorting: GroupSort[]) => {
  const [sort] = sorting;

  if (!sort || !isColumnId(sort.id)) {
    return groups;
  }

  const getValue = GROUP_SORT_VALUES[sort.id];

  return [...groups].sort((a, b) =>
    compareSortValues(getValue(a), getValue(b), sort.desc)
  );
};
