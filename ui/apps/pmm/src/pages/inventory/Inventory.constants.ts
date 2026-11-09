import type { SvgIconProps } from '@mui/material/SvgIcon';
import {
  HaproxyIcon,
  PerconaMoIcon,
  PerconaMyIcon,
  PerconaPoIcon,
  PerconaVaIcon,
  ProxyIcon,
} from '@percona/peak-ui';
import type { UseTableUrlStateOptions } from 'hooks/utils/useTableUrlState';
import { type MRT_RowData, type MRT_TableOptions } from 'material-react-table';
import type { FC } from 'react';
import { ManagedServiceType, ServiceStatus } from 'types/services.types';
import { type InventoryEntity, MonitoringStatus } from './Inventory.types';

// Same polling rate the Grafana inventory pages used
export const INVENTORY_REFETCH_INTERVAL = 5000;

export const INVENTORY_PAGE_SIZE = 25;

export const INVENTORY_URL_PREFIX: Record<InventoryEntity, string> = {
  services: 'services',
  nodes: 'nodes',
  agents: 'agents',
};

export const getUrlStateOptions = (
  entity: InventoryEntity
): UseTableUrlStateOptions => ({
  paramPrefix: INVENTORY_URL_PREFIX[entity],
  defaults: { pagination: { pageIndex: 0, pageSize: INVENTORY_PAGE_SIZE } },
});

export const INVENTORY_TABLE_OPTIONS = {
  enableStickyHeader: true,
  // Sorting, filters and columns stay reachable from the headers and toolbar,
  // while the header space goes to the column names
  enableColumnActions: false,
  positionActionsColumn: 'last',
  positionToolbarAlertBanner: 'none',
  globalFilterFn: 'contains',
} satisfies Partial<MRT_TableOptions<MRT_RowData>>;

// Persists the services' cluster view, under the key Grafana used
export const CLUSTERS_SWITCH_KEY = 'pmm-organize-by-clusters';

export const SERVICE_TYPE_ICONS: Partial<
  Record<ManagedServiceType, FC<SvgIconProps>>
> = {
  [ManagedServiceType.mysql]: PerconaMyIcon,
  [ManagedServiceType.mongodb]: PerconaMoIcon,
  [ManagedServiceType.postgresql]: PerconaPoIcon,
  [ManagedServiceType.valkey]: PerconaVaIcon,
  [ManagedServiceType.proxysql]: ProxyIcon,
  [ManagedServiceType.haproxy]: HaproxyIcon,
};

// External services have no dashboard of their own
export const SERVICE_DASHBOARDS: Partial<Record<ManagedServiceType, string>> = {
  [ManagedServiceType.mysql]: 'mysql-instance-summary/mysql-instance-summary',
  [ManagedServiceType.mongodb]:
    'mongodb-instance-summary/mongodb-instance-summary',
  [ManagedServiceType.postgresql]:
    'postgresql-instance-summary/postgresql-instance-summary',
  [ManagedServiceType.proxysql]:
    'proxysql-instance-summary/proxysql-instance-summary',
  [ManagedServiceType.haproxy]:
    'haproxy-instance-summary/haproxy-instance-summary',
  [ManagedServiceType.valkey]: 'valkey-overview/valkey-redis-overview',
};

export const QAN_SERVICE_TYPES = [
  ManagedServiceType.mysql,
  ManagedServiceType.mongodb,
  ManagedServiceType.postgresql,
];

export const SERVICE_STATUSES = [
  ServiceStatus.up,
  ServiceStatus.down,
  ServiceStatus.unknown,
  ServiceStatus.unspecified,
];

export const SERVICE_STATUS_OPTIONS = [
  { label: 'Up', value: ServiceStatus.up },
  { label: 'Down', value: ServiceStatus.down },
  { label: 'Unknown', value: ServiceStatus.unknown },
  { label: 'N/A', value: ServiceStatus.unspecified },
];

export const MONITORING_STATUS_OPTIONS = [
  { label: MonitoringStatus.ok, value: MonitoringStatus.ok },
  { label: MonitoringStatus.warning, value: MonitoringStatus.warning },
  { label: MonitoringStatus.failed, value: MonitoringStatus.failed },
];

// Filters of the former Grafana inventory pages, kept working for existing
// links and bookmarks; each other parameter named a filtered column.
export const LEGACY_SEARCH_TEXT_PARAM = 'search-text-input';
export const LEGACY_SEARCH_COLUMN_PARAM = 'search-select';
export const LEGACY_ALL_COLUMNS = 'All';
export const LEGACY_FILTER_PARAMS = [
  'status',
  'agentsStatus',
  'nodeType',
  'type',
  'isPmmServerNode',
];
// Grafana mapped services without a status to 'N/A'
export const LEGACY_FILTER_VALUES: Record<string, string> = {
  'N/A': ServiceStatus.unspecified,
};
