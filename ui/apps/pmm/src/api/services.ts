import type {
  AddAzureDatabasePayload,
  AddServicePayload,
  DiscoverAzurePayload,
  DiscoverAzureResponse,
  DiscoverRDSPayload,
  DiscoverRDSResponse,
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
  ManagedServicesResponse,
} from 'types/services.types';
import { api } from './api';
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
  const res = await api.get('/management/services', { params });
  return res.data;
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
  payload: DiscoverRDSPayload
): Promise<DiscoverRDSResponse> => {
  const res = await api.post<DiscoverRDSResponse>(
    '/management/services:discoverRDS',
    payload,
    { disableNotifications: true }
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
