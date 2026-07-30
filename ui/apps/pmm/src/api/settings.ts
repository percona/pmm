import type { AxiosRequestConfig } from 'axios';
import {
  GetFrontendSettingsResponse,
  GetReadonlySettingsResponse,
  GetSettingsResponse,
  ReadonlySettings,
  Settings,
  UpdateSettingsPayload,
} from 'types/settings.types';
import { api, grafanaApi } from './api';

export const getSettings = async (config?: AxiosRequestConfig) => {
  const res = await api.get<GetSettingsResponse>('/server/settings', config);
  return res.data.settings;
};

export const getReadonlySettings = async (): Promise<ReadonlySettings> => {
  const res = await api.get<GetReadonlySettingsResponse>(
    '/server/settings/readonly'
  );
  return res.data.settings;
};

export const getFrontendSettings = async () => {
  const res =
    await grafanaApi.get<GetFrontendSettingsResponse>('/frontend/settings');
  return res.data;
};

export const updateSettings = async (
  payload: UpdateSettingsPayload
): Promise<Settings> => {
  const res = await api.put<GetSettingsResponse>('/server/settings', payload);
  return res.data.settings;
};
