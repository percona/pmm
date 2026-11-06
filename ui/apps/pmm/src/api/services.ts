import type {
  AddAzureDatabasePayload,
  AddServicePayload,
  BaseService,
  ChangeServiceParams,
  DiscoverAzurePayload,
  DiscoverAzureResponse,
  DiscoverRDSPayload,
  DiscoverRDSResponse,
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
  ManagedServicesResponse,
  RemoveServiceParams,
} from 'types/services.types';
import { api } from './api';
import {
  type GetServicePayload,
  type ListServicesPayload,
  getIfExists,
  RAW_REQUEST,
  RAW_RESPONSE,
  toBaseService,
  toManagedService,
} from './inventory.utils';
import { withVerbatimKeys } from './api.utils';

const MAP_FIELDS = ['customLabels', 'extraDsnParams'];

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

export const getService = async (
  serviceId: string
): Promise<BaseService | null> => {
  const payload = await getIfExists<GetServicePayload>(
    `/inventory/services/${serviceId}`
  );
  return payload && toBaseService(Object.values(payload)[0]);
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

export const addService = async (payload: AddServicePayload): Promise<void> => {
  await api.post('/management/services', withVerbatimKeys(payload, MAP_FIELDS));
};

export const addAzureService = async (
  payload: AddAzureDatabasePayload
): Promise<void> => {
  await api.post(
    '/management/services/azure',
    withVerbatimKeys(payload, MAP_FIELDS)
  );
};

// The RDS discovery page reports errors itself, rewording the missing-credentials one.
export const discoverRDS = async (
  payload: DiscoverRDSPayload,
  signal?: AbortSignal
): Promise<DiscoverRDSResponse> => {
  const res = await api.post<DiscoverRDSResponse>(
    '/management/services:discoverRDS',
    payload,
    { disableNotifications: true, signal }
  );
  return res.data;
};

export const discoverAzure = async (
  payload: DiscoverAzurePayload
): Promise<DiscoverAzureResponse> => {
  const res = await api.post<DiscoverAzureResponse>(
    '/management/services:discoverAzure',
    payload
  );
  return res.data;
};
