import { ManagedServicesResponse } from 'types/services.types';

export const shouldShowAddService = (
  data?: ManagedServicesResponse
): boolean => {
  if (!data?.services) {
    return false;
  }

  // Count services by type (API returns lowercase type names like "mysql", "mongodb")
  const countByType = data.services.reduce(
    (acc, service) => {
      const type = service.serviceType.toLowerCase();
      acc[type] = (acc[type] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const minRequiredByType: Record<string, number> = {
    external: 1,
    haproxy: 1,
    mongodb: 1,
    mysql: 1,
    // Take into account the default PostgreSQL service
    postgresql: 2,
    proxysql: 1,
    valkey: 1,
  };

  return Object.entries(minRequiredByType).every(
    ([type, min]) => (countByType[type] ?? 0) < min
  );
};
