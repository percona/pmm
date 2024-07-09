import { GetSettingsResponse } from 'types/settings.types';
import { api } from './api';
import { AxiosResponse } from 'axios';

export const getSettings = async () => {
  const res = await api.post<void, AxiosResponse<GetSettingsResponse>>(
    'Settings/Get'
  );
  return res.data;
};
