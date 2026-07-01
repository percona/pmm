export interface ReadonlySettings {
  updatesEnabled: boolean;
  telemetryEnabled: boolean;
  advisorEnabled: boolean;
  alertingEnabled: boolean;
  nativeQanEnabled: boolean;
  pmmPublicAddress: string;
  backupManagementEnabled: boolean;
  azurediscoverEnabled: boolean;
  enableAccessControl: boolean;
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
  updateSnoozeDuration?: string;
  metricsResolutions?: MetricsResolutions;
  dataRetention?: string;
  sshKey?: string;
  awsPartitions?: string[];
  advisorRunIntervals?: AdvisorRunIntervals;
  telemetrySummaries?: string[];
  enableInternalPgQan?: boolean;
  defaultRoleId?: number;
  otel?: OtelSettings;
}

export interface OtelSettings {
  collectorEnabled?: boolean;
  collector_enabled?: boolean;
  logsRetentionDays?: number;
  logs_retention_days?: number;
  tracesRetentionDays?: number;
  traces_retention_days?: number;
  metricsRetentionDays?: number;
  metrics_retention_days?: number;
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
  enableNativeQan?: boolean;
  awsPartitions?: string[];
  updateSnoozeDuration?: string;
  otel?: {
    collectorEnabled?: boolean;
    logsRetentionDays?: number;
    tracesRetentionDays?: number;
    metricsRetentionDays?: number;
  };
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
