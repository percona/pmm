export interface ReadonlySettingsPayload {
  updates_enabled?: boolean;
  advisor_enabled?: boolean;
}

export interface ReadonlySettingsResponse {
  settings?: ReadonlySettingsPayload;
}

export interface ReadonlySettings {
  updatesEnabled: boolean;
  advisorEnabled: boolean;
}
