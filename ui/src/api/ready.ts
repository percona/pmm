import { api } from './api';

export const getReadiness = async () => {
  const res = await api.get<Record<string, never>>('/server/readyz');
  return res.data;
};
