export interface GetUpdatesParams {
  force: boolean;
  onlyInstalledVersion?: boolean;
}

export interface CurrentInfo {
  version: string;
  fullVersion: string;
  timestamp: string | null;
}

export interface LatestInfo {
  version: string;
  tag: string;
  timestamp: string | null;
  releaseNotesText: string;
  releaseNotesUrl: string;
}

export interface GetUpdatesResponse {
  lastCheck: string;
  latest: LatestInfo;
  installed: CurrentInfo;
  latestNewsUrl: string;
  updateAvailable: boolean;
}

export interface StartUpdateBody {
  newImage?: string;
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

export enum UpdateStatus {
  Pending = 'pending',
  Updating = 'updating',
  Restarting = 'restarting',
  Completed = 'completed',
  Error = 'error',
  Checking = 'checking',
  UpToDate = 'up-to-date',
  UpdateClients = 'update-clients',
}

export interface GetChangeLogItem {
  version: string;
  tag: string;
  timestamp: string;
  releaseNotesUrl: string;
  releaseNotesText: string;
}

export interface GetChangeLogsResponse {
  updates: GetChangeLogItem[];
  lastCheck: string;
}
