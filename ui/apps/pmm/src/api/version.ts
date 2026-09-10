import { VersionResponse } from 'types/version.types';
import { api } from './api';

export const getVersion = async (): Promise<VersionResponse> => {
  const res = await api.get<VersionResponse>('/server/version');
  return res.data;
};
