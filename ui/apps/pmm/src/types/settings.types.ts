export interface ReadonlySettings {
  updatesEnabled: boolean;
  telemetryEnabled: boolean;
  advisorEnabled: boolean;
  alertingEnabled: boolean;
  pmmPublicAddress: string;
  backupManagementEnabled: boolean;
  azurediscoverEnabled: boolean;
  enableAccessControl: boolean;
}

/** Settings the server will refuse to change. Values match the SettingName proto enum. */
export enum SettingName {
  unspecified = 'SETTING_NAME_UNSPECIFIED',
  dataRetention = 'SETTING_NAME_DATA_RETENTION',
  updatesEnabled = 'SETTING_NAME_UPDATES_ENABLED',
  telemetryEnabled = 'SETTING_NAME_TELEMETRY_ENABLED',
  alertingEnabled = 'SETTING_NAME_ALERTING_ENABLED',
  azurediscoverEnabled = 'SETTING_NAME_AZUREDISCOVER_ENABLED',
  metricsResolutions = 'SETTING_NAME_METRICS_RESOLUTIONS',
  enableInternalPgQan = 'SETTING_NAME_ENABLE_INTERNAL_PG_QAN',
}

/** Why a setting cannot be changed. Values match the LockReason proto enum. */
export enum LockReason {
  unspecified = 'LOCK_REASON_UNSPECIFIED',
  environment = 'LOCK_REASON_ENVIRONMENT',
  highAvailability = 'LOCK_REASON_HIGH_AVAILABILITY',
}

export interface SettingLock {
  setting: SettingName;
  reason: LockReason;
  /** Only set when reason is LockReason.environment. */
  environmentVariable?: string;
}

export interface MetricsResolutions {
  hr: string;
  mr: string;
  lr: string;
}

export interface AdvisorRunIntervals {
  rareInterval: string;
  standardInterval: string;
  frequentInterval: string;
}

export interface GetReadonlySettingsResponse {
  settings: ReadonlySettings;
}

export interface Settings extends ReadonlySettings {
  metricsResolutions?: MetricsResolutions;
  dataRetention?: string;
  sshKey?: string;
  awsPartitions?: string[];
  advisorRunIntervals?: AdvisorRunIntervals;
  telemetrySummaries?: string[];
  enableInternalPgQan?: boolean;
  defaultRoleId?: number;
  lockedSettings?: SettingLock[];
}

/** Payload for PUT /server/settings - partial updates supported */
export interface UpdateSettingsPayload {
  sshKey?: string;
  metricsResolutions?: MetricsResolutions;
  dataRetention?: string;
  pmmPublicAddress?: string;
  enableTelemetry?: boolean;
  enableAlerting?: boolean;
  enableAdvisor?: boolean;
  advisorRunIntervals?: AdvisorRunIntervals;
  enableBackupManagement?: boolean;
  enableAzurediscover?: boolean;
  enableUpdates?: boolean;
  enableAccessControl?: boolean;
  enableInternalPgQan?: boolean;
  awsPartitions?: string[];
}

export interface FrontendSettings extends GetFrontendSettingsResponse {}

export interface GetSettingsResponse {
  settings: Settings;
}

export interface GetFrontendSettingsResponse {
  anonymousEnabled: boolean;
  appSubUrl: string;
  apps: Record<string, GrafanaApp>;
  buildInfo: GrafanaBuildInfo;
  exploreEnabled: boolean;
  featureToggles: {
    exploreMetrics: boolean;
  };
  unifiedAlertingEnabled: boolean;
  disableLoginForm: boolean;
  auth: {
    disableLogin: boolean;
  };
}

export interface GrafanaBuildInfo {
  version: string;
  versionString: string;
}

export interface GrafanaApp {
  id: string;
  preload: boolean;
}
