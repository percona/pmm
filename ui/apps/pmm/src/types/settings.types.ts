// doesn't yet have the complete response
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

export interface MetricsResolutions {
  hr: string;
  mr: string;
  lr: string;
}

export interface AdvisorRunIntervalsSettings {
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
  advisorRunIntervals?: AdvisorRunIntervalsSettings;
  telemetrySummaries?: string[];
  enableInternalPgQan?: boolean;
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
  advisorRunIntervals?: {
    rareInterval: string;
    standardInterval: string;
    frequentInterval: string;
  };
  enableBackupManagement?: boolean;
  enableAzurediscover?: boolean;
  enableUpdates?: boolean;
  enableAccessControl?: boolean;
  enableInternalPgQan?: boolean;
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
