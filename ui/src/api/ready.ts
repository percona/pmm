import { AxiosResponse } from 'axios';
import { api } from './api';

export const getReadiness = async () => {
  const res = await api.get<void, AxiosResponse<Record<string, never>>>(
    '/readyz'
  );
  return res.data;
};
