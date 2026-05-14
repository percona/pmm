import {
  GRAFANA_SUB_PATH,
  PMM_LOGIN_URL,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import { Path } from 'react-router-dom';

/** True when the PMM URL (pathname + optional search/hash) is the Grafana login route. */
export const isGrafanaLoginPath = (pathnameSearchHash: string) => {
  const path = pathnameSearchHash.split(/[?#]/)[0].replace(/\/$/, '') || '/';
  return path === PMM_LOGIN_URL || path.endsWith(PMM_LOGIN_URL);
};

export const getLocationUrl = (location: Path) => {
  const { pathname: rawPath, search, hash } = location;

  const pathname = rawPath.startsWith('/pmm-ui')
    ? rawPath.replace('/pmm-ui', PMM_NEW_NAV_PATH)
    : rawPath.startsWith(GRAFANA_SUB_PATH)
      ? rawPath
      : PMM_NEW_NAV_GRAFANA_PATH + rawPath;

  return pathname + search + hash;
};
