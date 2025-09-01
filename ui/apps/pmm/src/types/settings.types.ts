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

export interface GetReadonlySettingsResponse {
  settings: ReadonlySettings;
}

export interface Settings extends ReadonlySettings {}

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
