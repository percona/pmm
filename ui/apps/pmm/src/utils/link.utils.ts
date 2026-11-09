export const constructUrl = (location: {
  pathname: string;
  search: string;
  hash: string;
}) => location.pathname + location.search + location.hash;

// Same params Grafana's keepTime ($__url_time_range) puts on a dashboard link
const TIME_RANGE_PARAMS = ['from', 'to', 'timezone'];

// Carries the current dashboard's time range over to a dashboard link that doesn't set one yet
export const keepTimeRange = (
  url: string,
  location: { pathname: string; search: string }
) => {
  if (!url.includes('/d/') || !location.pathname.includes('/d/')) {
    return url;
  }

  const linkParams = new URLSearchParams(url.split('?')[1]);
  const currentParams = new URLSearchParams(location.search);
  const missing = new URLSearchParams();

  for (const param of TIME_RANGE_PARAMS) {
    const value = currentParams.get(param);

    if (value && !linkParams.has(param)) {
      missing.set(param, value);
    }
  }

  const search = missing.toString();

  if (!search) {
    return url;
  }

  return url + (url.includes('?') ? '&' : '?') + search;
};

export const createRealtimeOverviewUrl = (serviceIds: string[]) => {
  const params = new URLSearchParams();
  serviceIds.forEach((serviceId) => params.append('serviceIds', serviceId));
  return `/rta/overview?${params.toString()}`;
};

export const createRealtimeSessionsUrl = (serviceIds: string[]) => {
  const params = new URLSearchParams();
  serviceIds.forEach((serviceId) => params.append('serviceIds', serviceId));
  return `/rta/sessions?fromOverview=true&${params.toString()}`;
};
