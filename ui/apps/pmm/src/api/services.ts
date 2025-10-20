import {
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
} from 'types/services.types';
import { api } from './api';

export const getServiceTypes = async (): Promise<ListTypesResponse> => {
  const res = await api.post('/inventory/services:getTypes');
  return res.data;
};

export const listServices = async (
  params: ListServicesParams
): Promise<ListServicesResponse> => {
  const res = await api.get('/inventory/services', { params: params });
  return res.data;
};
