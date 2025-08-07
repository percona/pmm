import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';

export const getLocationUrl = (location: Location) => {
  const pathname = location.pathname.startsWith('/pmm-ui')
    ? location.pathname.replace('/pmm-ui', PMM_NEW_NAV_PATH)
    : PMM_NEW_NAV_GRAFANA_PATH + location.pathname;

  return pathname + location.search + location.hash;
};
