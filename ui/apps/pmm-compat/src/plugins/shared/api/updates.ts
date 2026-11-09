import { get } from './api';
import type {
  CheckUpdatesParams,
  CheckUpdatesResponse,
  UpdatesInfo,
} from '../types/updates.types';

const toUpdatesInfo = (response: CheckUpdatesResponse): UpdatesInfo => ({
  installed: response.installed
    ? {
        version: response.installed.version,
        fullVersion: response.installed.full_version,
        timestamp: response.installed.timestamp,
      }
    : undefined,
  latest: response.latest
    ? {
        version: response.latest.version,
        tag: response.latest.tag,
        timestamp: response.latest.timestamp,
      }
    : undefined,
  lastChecked: response.last_check,
  latestNewsUrl: response.latest_news_url,
  updateAvailable: !!response.update_available,
});

export const checkForUpdates = async (
  params: CheckUpdatesParams = { force: false }
): Promise<UpdatesInfo> =>
  toUpdatesInfo(await get<CheckUpdatesResponse>('server/updates', params));
