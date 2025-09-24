import {
  GetFrontendSettingsResponse,
  GetSettingsResponse,
} from 'types/settings.types';
import { api, grafanaApi } from './api';
import { parseDuration } from 'utils/duration';

export const getSettings = async () => {
  const res = await api.get<GetSettingsResponse>('/server/settings');
  return {
    ...res.data.settings,
    updatesSnoozeDurationMs: parseDuration(
      res.data.settings.updatesSnoozeDuration
    ),
  };
};

export const getReadonlySettings = async () => {
  const res = await api.get<GetSettingsResponse>('/server/settings/readonly');
  return res.data.settings;
};

export const getFrontendSettings = async () => {
  const res =
    await grafanaApi.get<GetFrontendSettingsResponse>('/frontend/settings');
  return res.data;
};
