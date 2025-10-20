import { ListServicesResponse } from 'types/services.types';

export const shouldShowAddService = (
  services?: ListServicesResponse
): boolean => {
  if (!services) {
    return false;
  }

  const minRequiredByService: Record<keyof ListServicesResponse, number> = {
    external: 1,
    haproxy: 1,
    mongodb: 1,
    mysql: 1,
    // Take into account the default postgresql service
    postgresql: 2,
    proxysql: 1,
    valkey: 1,
  };

  return (
    Object.entries(minRequiredByService) as [
      keyof ListServicesResponse,
      number,
    ][]
  ).every(([key, min]) => services[key].length < min);
};
