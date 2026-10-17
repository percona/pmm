import type {
  ChangeServiceParams,
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
  ManagedServicesResponse,
  RemoveServiceParams,
} from 'types/services.types';
import { api } from './api';
import {
  type ListServicesPayload,
  RAW_REQUEST,
  RAW_RESPONSE,
  toManagedService,
} from './inventory.utils';

export const getServiceTypes = async (): Promise<ListTypesResponse> => {
  const res = await api.post('/inventory/services:getTypes');
  return res.data;
};

export const listServices = async (
  params: ListServicesParams
): Promise<ListServicesResponse> => {
  const res = await api.get('/inventory/services', { params });
  return res.data;
};

export const listManagedServices = async (
  params: ListServicesParams
): Promise<ManagedServicesResponse> => {
  const res = await api.get<ListServicesPayload>('/management/services', {
    params,
    ...RAW_RESPONSE,
  });
  return { services: (res.data.services ?? []).map(toManagedService) };
};

export const removeService = async ({
  serviceId,
  force,
}: RemoveServiceParams) => {
  await api.delete(`/inventory/services/${serviceId}`, { params: { force } });
};

export const changeService = async ({
  serviceId,
  environment,
  cluster,
  replicationSet,
  customLabels,
}: ChangeServiceParams) => {
  await api.put(
    `/inventory/services/${serviceId}`,
    {
      environment,
      cluster,
      replication_set: replicationSet,
      custom_labels: customLabels && { values: customLabels },
    },
    RAW_REQUEST
  );
};
