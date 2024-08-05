export interface GetUpdatesParams {
  force: boolean;
  onlyInstalledVersion?: boolean;
}

export interface CurrentInfo {
  version?: string;
  full_version?: string;
  timestamp?: string;
}

export interface LatestInfo {
  version?: string;
  tag?: string;
  timestamp?: string;
}

export interface GetUpdatesResponse {
  lastCheck: string;
  latest?: LatestInfo;
  installed: CurrentInfo;
  latestNewsUrl?: string;
  updateAvailable?: boolean;
}

export interface StartUpdateResponse {
  authToken: string;
  logOffset: number;
}

export interface GetUpdateStatusBody {
  authToken: string;
  logOffset: number;
}

export interface GetUpdateStatusResponse {
  done: boolean;
  logOffset: number;
  logLines: string[];
}
