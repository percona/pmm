export interface GetUpdatesBody {
  force: boolean;
  onlyInstalledVersion?: boolean;
}

export interface GetUpdatesResponse {
  lastCheck: string;
  latest: {
    fullVersion: string;
    timestamp: string;
    version: string;
  };
  installed: {
    fullVersion: string;
    timestamp: string;
    version: string;
  };
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
