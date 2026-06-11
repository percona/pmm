import { EmptyResponse } from 'types/util.types';
import { api } from './api';

export const getReadiness = async () => {
  const res = await api.get<EmptyResponse>('/server/readyz');
  return res.data;
};
