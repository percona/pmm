export interface CheckUpdatesParams {
  force: boolean;
  only_installed_version?: boolean;
}

export interface CheckUpdatesResponse {
  installed?: {
    version?: string;
    full_version?: string;
    timestamp?: string;
  };
  latest?: {
    version?: string;
    tag?: string;
    timestamp?: string;
  };
  update_available?: boolean;
  latest_news_url?: string;
  last_check?: string;
}

export interface CurrentInformation {
  version?: string;
  fullVersion?: string;
  timestamp?: string;
}

export interface LatestInformation {
  version?: string;
  tag?: string;
  timestamp?: string;
}

export interface UpdatesInfo {
  installed?: CurrentInformation;
  latest?: LatestInformation;
  updateAvailable: boolean;
  latestNewsUrl?: string;
  lastChecked?: string;
}
