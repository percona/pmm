export interface ServerVersionInfo {
  version: string;
  fullVersion?: string;
  timestamp?: string | null;
}

export interface GetServerVersionResponse {
  version: string;
  server?: ServerVersionInfo;
  managed?: ServerVersionInfo;
}
