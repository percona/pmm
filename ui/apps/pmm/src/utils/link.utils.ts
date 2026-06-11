export const constructUrl = (location: {
  pathname: string;
  search: string;
  hash: string;
}) => location.pathname + location.search + location.hash;

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
