import { GetSettingsResponse } from 'types/settings.types';
import { api } from './api';

export const getSettings = async () => {
  const res = await api.get<GetSettingsResponse>('/server/settings');
  return res.data.settings;
};
