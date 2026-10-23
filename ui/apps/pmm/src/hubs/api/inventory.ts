import { api } from 'api/api';
import type {
  ManagedServicesResponse,
  ServiceType,
} from 'types/services.types';

/** `GET /v1/management/services` for one type, without the error toast: the hub shows a failed source in its own section. */
export const listManagedServicesQuietly = async (serviceType: ServiceType) => {
  const res = await api.get<ManagedServicesResponse>('/management/services', {
    params: { serviceType },
    disableNotifications: true,
  });

  return res.data.services ?? [];
};
