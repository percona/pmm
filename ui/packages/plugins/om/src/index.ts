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

export { OmApp } from './OmApp';
export { omBase, useOmBase } from './useOmBase';
export { OverviewPage } from './OverviewPage';
export { ServicesPage } from './ServicesPage';
export { HostsPage } from './HostsPage';
export { InventoryPage } from './InventoryPage';
export { StatusBadge, RunStatusBadge } from './components/HealthBadge';
export { Duration, Percent } from './components/Metric';
export { RunEntities } from './components/RunEntities';
export { ProbeValue } from './components/ProbeValue';
export { ConfigForm } from './components/ConfigForm';
export { SnapshotBar } from './components/SnapshotBar';
export { SyncButton } from './components/SyncButton';
export { OmHeader } from './components/OmHeader';
export { Unavailable } from './components/Unavailable';
export {
  useOmTopology,
  toClusterRows,
  toEnvironmentSections,
  toServiceRows,
  useOmTopologyRuns,
  useTriggerOmTopologyRun,
  useInvalidateOmTopologySnapshot,
  isRunActive,
  OM_TOPOLOGY_RUNS_LIMIT,
} from './hooks';
export {
  OM_ROUTE_OVERVIEW,
  OM_ROUTE_SERVICES,
  OM_ROUTE_HOSTS,
  OM_ROUTE_INVENTORY,
  HOST_DATABASE_STATE_LABEL,
  HOST_DATABASE_STATE_COLOR,
  SETTING_LABEL,
  SETTING_UNIT,
  SETTING_HELP,
  SERVICE_STATUS_LABEL,
  SERVICE_STATUS_COLOR,
  PROCESS_ROLE_LABEL,
  UNAVAILABLE_PHRASE,
  RUN_STATUS_LABEL,
  RUN_STATUS_COLOR,
} from './constants';
export {
  useOmInventoryHosts,
  useOmInventoryServices,
  useOmInventoryRuns,
  useOmInventoryRun,
  useOmInventoryConfig,
  useUpdateOmInventoryConfig,
  useResetOmInventoryConfig,
  useRefreshInventory,
  useForgetHost,
  useForgetService,
} from './inventoryHooks';
export type { OmHostFilters } from './inventoryHooks';
export {
  ageSeconds,
  databaseState,
  isFailing,
  joinServiceInventory,
  missingRowReason,
  repoReachability,
  toHostRows,
} from './inventory';
export type { OmEstateStatus } from './inventory';
export {
  formatAge,
  formatDuration,
  formatRunDuration,
  runDurationSeconds,
  formatTimestamp,
} from './format';
export type {
  OmCluster,
  OmClusterRow,
  OmClusterType,
  OmEnvironment,
  OmEnvironmentSection,
  OmProcessRole,
  OmService,
  OmServiceRow,
  OmServiceStatus,
  OmTopologyResponse,
  OmTopologyRun,
  OmTopologyRunAccepted,
  OmTopologyRunCounts,
  OmTopologyRunError,
  OmTopologyRunStatus,
  OmTopologySnapshotEnvelope,
  OmTopologySummary,
  OmUnavailableReason,
} from './types';
// The inventory contract, alongside the hooks that return it. Without these a consumer
// can call useOmInventoryHosts but cannot name what it hands back, which the topology
// half of this file has never had to live with.
export type {
  OmExecutorResolution,
  OmHostDatabaseState,
  OmHostRow,
  OmInventoryExecutor,
  OmInventoryFreshness,
  OmInventoryHost,
  OmInventoryRun,
  OmInventoryRunAccepted,
  OmInventoryRunCounts,
  OmInventoryRunDetail,
  OmInventoryRunEntity,
  OmInventoryRunEntityService,
  OmInventoryService,
  OmInventorySetting,
  OmRepoReachability,
  OmServiceInventoryRow,
  OmSettingReload,
  OmUnregisteredMongod,
} from './types';
