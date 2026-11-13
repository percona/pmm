/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import type {
  MRT_SortingState,
  MRT_VisibilityState,
} from 'material-react-table';
import { NO_GROUPING, type GroupByOption } from '../components/GroupBySelect';
import type { FilterOption } from '../components/MultiSelectFilter';
import { CLUSTER_HEALTH_LABEL, CLUSTER_TYPE_LABEL } from '../constants';
import type { OmClusterHealth, OmClusterType } from '../types';
import type { TableUrlStateOptions } from '../useTableUrlState';
import { Messages } from './FleetClustersTab.messages';
import type { ClusterGrouping } from './FleetClustersTab.types';

export const CLUSTERS_TABLE_NAME = 'om-fleet-clusters';

export const GROUPING_STORAGE_KEY = 'om.fleet.clusters.grouping';

const GROUPINGS: ClusterGrouping[] = [
  'health',
  'env_name',
  'type',
  'versions',
  NO_GROUPING,
];

export const GROUP_BY_OPTIONS: GroupByOption[] = GROUPINGS.map((value) => ({
  value,
  label: Messages.groupBy[value],
}));

/** Version groups' values for a cluster on several versions, or on none known. */
export const MIXED_VERSIONS = '(mixed)';
export const UNKNOWN_VERSION = '(unknown)';

/** Type groups lead with the cluster that has the most moving parts. */
export const TYPE_GROUP_ORDER: OmClusterType[] = [
  'CLUSTER_TYPE_SHARDED',
  'CLUSTER_TYPE_REPLICA_SET',
  'CLUSTER_TYPE_STANDALONE',
];

/** Hidden to fit 1440px with the nav open; all of them are a click away in the chooser. */
export const HIDDEN_BY_DEFAULT: MRT_VisibilityState = {
  total_services: false,
  roles: false,
  max_cpu_percent: false,
  min_connections_free_percent: false,
};

/** Replication lag at or above this reads as an error; to confirm with design. */
export const LAG_ALERT_SECONDS = 30;

/** An oplog window below this reads as a warning; to confirm with design. */
export const OPLOG_WARN_SECONDS = 60 * 60;

/** Worst first, the order the health filter offers them in. */
export const HEALTH_FILTER_VALUES: OmClusterHealth[] = [
  'down',
  'degraded',
  'unknown',
  'healthy',
];

/** How each cluster type is spelled in the URL. */
export const TYPE_FILTER_VALUE: Partial<Record<OmClusterType, string>> = {
  CLUSTER_TYPE_REPLICA_SET: 'replica-set',
  CLUSTER_TYPE_SHARDED: 'sharded',
  CLUSTER_TYPE_STANDALONE: 'standalone',
};

/** The unnamed environment's filter value: peak-ui drops an empty one as no filter. */
export const UNNAMED_ENVIRONMENT_VALUE = '(none)';

export const HEALTH_OPTIONS: FilterOption[] = HEALTH_FILTER_VALUES.map(
  (value) => ({ value, label: CLUSTER_HEALTH_LABEL[value] })
);

export const TYPE_OPTIONS: FilterOption[] = Object.entries(
  TYPE_FILTER_VALUE
).map(([type, value]) => ({
  value,
  label: CLUSTER_TYPE_LABEL[type as OmClusterType] ?? value,
}));

/** Trouble first within each group; being the default, it stays out of the URL. */
export const DEFAULT_SORTING: MRT_SortingState = [
  { id: 'health', desc: false },
  { id: 'cluster_name', desc: false },
];

export const URL_STATE: TableUrlStateOptions = {
  paramPrefix: 'clusters',
  defaults: { sorting: DEFAULT_SORTING },
  sync: { pagination: false },
};
