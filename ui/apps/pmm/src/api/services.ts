import { ListTypesResponse } from 'types/services.types';
import { api } from './api';

export const getServiceTypes = async (): Promise<ListTypesResponse> => {
  const res = await api.post('/inventory/services:getTypes');
  return res.data;
};
