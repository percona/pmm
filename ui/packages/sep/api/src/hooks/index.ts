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

export { useCurrentUser } from './useCurrentUser';
export { usePluginSchema } from './usePluginSchema';
export {
  usePluginTasks,
  usePluginTask,
  useCreatePluginTask,
  useUpdatePluginTask,
  usePluginEntityList,
  usePluginEntityDetail,
  useCreatePluginEntity,
  useUpdatePluginEntity,
  useDeletePluginEntity,
  useDeletePluginTask,
  normalizePluginListResponse,
  fetchAllPluginListPages,
  DEFAULT_PLUGIN_LIST_OFFSET,
  DEFAULT_PLUGIN_LIST_LIMIT,
  RUNNING_STATUSES,
  isRunningStatus,
} from './usePluginTasks';
export type {
  PluginListPagination,
  PluginListQueryOptions,
  PluginListResult,
  PaginatedPluginList,
  TaskHistoryStatus,
} from './usePluginTasks';
export { useAlertConfig, ALERT_CONFIG_QUERY_KEY } from './useAlertConfig';
export type { AlertConfig } from './useAlertConfig';
export { useDashboardStats } from './useDashboardStats';
export type { DashboardStats } from './useDashboardStats';
export {
  useSettingsList,
  usePatchSetting,
  useResetSetting,
  settingErrorMessage,
  SETTINGS_QUERY_KEY,
  REDACTED_SECRET,
} from './useSettings';
export type {
  SettingClass,
  ReloadClassification,
  SettingResponse,
  SettingClassGroup,
  SettingsListResponse,
  SettingsPatch,
  PatchSettingVars,
  ResetSettingVars,
} from './useSettings';
export { useEnabledApps, ENABLED_APPS_QUERY_KEY } from './useEnabledApps';
export type { EnabledApp } from './useEnabledApps';
export {
  useAdminApps,
  useSetAppState,
  useForceDisableApp,
  isTransitional,
  appStateErrorMessage,
  ADMIN_APPS_QUERY_KEY,
  ADMIN_APP_MUTATION_KEY,
} from './useAdminApps';
export type {
  AdminApp,
  AppStateResult,
  AppLifecycleState,
  TransitionalState,
  SetAppStateVars,
  ForceDisableAppVars,
} from './useAdminApps';
export { useConfigExport } from './useConfigExport';
export { useAppInfo, APP_INFO_QUERY_KEY } from './useAppInfo';
export type { AppInfo } from './useAppInfo';
