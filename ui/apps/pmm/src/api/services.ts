import type {
  BaseService,
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

// Maps and lists are wrapped, so that an empty one reaches the server as a
// value that clears the stored one
const WRAPPED_CHANGES = new Set([
  'customLabels',
  'extraDsnParams',
  'disableCollectors',
  'statsCollections',
]);

const toSnakeCase = (name: string) =>
  name.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);

// Resolves with the server's warning, if any
export const updateService = async ({
  serviceId,
  serviceType,
  changes,
}: UpdateServiceParams): Promise<string> => {
  const params = Object.fromEntries(
    Object.entries(changes).map(([name, value]) => [
      toSnakeCase(name),
      WRAPPED_CHANGES.has(name) ? { values: value } : value,
    ])
  );
  const res = await api.put<{ warning?: string }>(
    `/management/services/${serviceId}`,
    { [serviceType]: params },
    RAW_REQUEST
  );

  return res.data.warning ?? '';
};
