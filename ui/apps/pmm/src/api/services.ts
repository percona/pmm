import type {
  AddAzureDatabasePayload,
  AddServicePayload,
  BaseService,
  DiscoverAzurePayload,
  DiscoverAzureResponse,
  DiscoverRDSPayload,
  DiscoverRDSResponse,
  EditableService,
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
  ManagedServiceType,
  ManagedServicesResponse,
  RemoveServiceParams,
  UpdateServiceParams,
} from 'types/services.types';
import { api } from './api';
import {
  type GetServicePayload,
  type ListAgentsPayload,
  type ListServicesPayload,
  getIfExists,
  RAW_REQUEST,
  RAW_RESPONSE,
  toBaseService,
  toInventoryService,
  toManagedService,
  toServiceAgent,
  toUpdateServicePayload,
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

export const getEditableService = async (
  serviceId: string
): Promise<EditableService | null> => {
  const payload = await getIfExists<GetServicePayload>(
    `/inventory/services/${serviceId}`
  );

  if (!payload) {
    return null;
  }

  const [serviceType, service] = Object.entries(payload)[0];
  const res = await api.get<ListAgentsPayload>('/management/agents', {
    params: { serviceId },
    ...RAW_RESPONSE,
  });

  return {
    service: toInventoryService(serviceType as ManagedServiceType, service),
    agents: (res.data.agents ?? []).map(toServiceAgent),
  };
};

// Resolves with the server's warning, if any
export const updateService = async ({
  serviceId,
  serviceType,
  changes,
}: UpdateServiceParams): Promise<string> => {
  const res = await api.put<{ warning?: string }>(
    `/management/services/${serviceId}`,
    { [serviceType]: toUpdateServicePayload(changes) },
    RAW_REQUEST
  );

  return res.data.warning ?? '';
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
