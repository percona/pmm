import {
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
  ManagedServicesResponse,
} from 'types/services.types';
import { api } from './api';

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
