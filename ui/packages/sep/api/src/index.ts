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

/// <reference path="./vite-env.d.ts" />

// Backend mount point
export { SEP_BASE_PATH } from './base';

// API client
export {
  apiClient,
  emitUnauthorized,
  getToken,
  refreshAccessToken,
  setTokenProvider,
  setTokenMinter,
  setOnUnauthorized,
  setOnRefreshed,
} from './client';
export type { MintedToken } from './client';

// Query client
export { createQueryClient, defaultQueryClientConfig } from './queryClient';

// Auth context (the provider lives in the host app; the context lives here so
// the framework and plugin packages can read it without depending on the host)
export {
  AuthContext,
  UNAUTHENTICATED_SESSION,
  deriveCanMutate,
  useAuth,
} from './auth-context';
export type { AuthSession, AuthState } from './auth-context';

// Errors
export {
  ApiError,
  normalizeAxiosError,
  normalizeBlobError,
  parseFieldErrors,
} from './errors';
export type {
  ApiErrorDetails,
  ApiErrorKind,
  FieldValidationError,
} from './errors';

// Auth
export {
  postLogin,
  postRefresh,
  postSession,
  postSessionExchange,
  postLogout,
  fetchCurrentUser,
} from './auth';

// Types (re-exported from generated OpenAPI schemas)
export type {
  OAuthTokenResponse,
  SessionExchangeTokenResponse,
  SPAOAuthTokenResponse,
  User,
} from './types/api';

// Generated OpenAPI type surfaces — use for typed openapi-fetch clients
// and for type-only imports in plugins and framework code.
export type {
  paths as MainPaths,
  components as MainComponents,
  operations as MainOperations,
} from './generated/main';
export type {
  paths as InventoryPaths,
  components as InventoryComponents,
} from './generated/inventory';
export type {
  paths as TasksPaths,
  components as TasksComponents,
} from './generated/tasks';
export type {
  paths as SepPaths,
  components as SepComponents,
} from './generated/sep';

// Typed request clients (openapi-fetch wrappers sharing interceptors with apiClient)
export { mainApi, sepApi, throwOnApiError } from './typed-client';

export type {
  PluginSchema,
  PluginEntitySchema,
  PluginField,
  SectionField,
  OneOfBranch,
  OneOfGroup,
  FormSection,
  ListColumn,
  ListView,
  DetailField,
  DetailSection,
  DetailView,
  PluginCapabilities,
  StringField,
  IntegerField,
  FloatField,
  BoolField,
  ChoiceField,
  ChoiceOption,
  MultiChoiceField,
  TextAreaField,
  DateTimeField,
  FileField,
  YamlField,
  ServiceField,
  SchemaField,
  TableField,
  HostField,
  RemoteChoiceField,
  ScriptPreviewField,
  Predicate,
  FieldGate,
  CardinalityRule,
  FailRule,
  RelatedApp,
} from './types/plugin-schema';

// Hooks
export {
  useCurrentUser,
  usePluginSchema,
  usePluginTasks,
  usePluginTask,
  RUNNING_STATUSES,
  isRunningStatus,
  useCreatePluginTask,
  useUpdatePluginTask,
  usePluginEntityList,
  usePluginEntityDetail,
  normalizePluginListResponse,
  DEFAULT_PLUGIN_LIST_OFFSET,
  DEFAULT_PLUGIN_LIST_LIMIT,
  useCreatePluginEntity,
  useUpdatePluginEntity,
  useDeletePluginEntity,
  useDeletePluginTask,
  useAlertConfig,
  ALERT_CONFIG_QUERY_KEY,
  useDashboardStats,
  useSettingsList,
  usePatchSetting,
  useResetSetting,
  settingErrorMessage,
  SETTINGS_QUERY_KEY,
  REDACTED_SECRET,
  useEnabledApps,
  ENABLED_APPS_QUERY_KEY,
  useAdminApps,
  useSetAppState,
  useForceDisableApp,
  isTransitional,
  appStateErrorMessage,
  ADMIN_APPS_QUERY_KEY,
  ADMIN_APP_MUTATION_KEY,
  useConfigExport,
  useAppInfo,
  APP_INFO_QUERY_KEY,
} from './hooks';
export type {
  AlertConfig,
  DashboardStats,
  EnabledApp,
  AppInfo,
  PluginListPagination,
  PluginListQueryOptions,
  PluginListResult,
  PaginatedPluginList,
  TaskHistoryStatus,
} from './hooks';
export type {
  AdminApp,
  AppStateResult,
  AppLifecycleState,
  TransitionalState,
  SetAppStateVars,
  ForceDisableAppVars,
} from './hooks';
export type {
  SettingClass,
  ReloadClassification,
  SettingResponse,
  SettingClassGroup,
  SettingsListResponse,
  SettingsPatch,
  PatchSettingVars,
  ResetSettingVars,
} from './hooks';
