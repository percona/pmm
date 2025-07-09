export interface Settings {
  updatesEnabled: boolean;
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
}

export interface GrafanaBuildInfo {
  version: string;
  versionString: string;
}

export interface GrafanaApp {
  id: string;
  preload: boolean;
}
