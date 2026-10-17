import { get } from './api';
import type {
  ReadonlySettings,
  ReadonlySettingsResponse,
} from '../types/settings.types';

export const getReadonlySettings = async (): Promise<ReadonlySettings> => {
  const { settings } = await get<ReadonlySettingsResponse>(
    'server/settings/readonly'
  );

  return {
    updatesEnabled: !!settings?.updates_enabled,
    advisorEnabled: !!settings?.advisor_enabled,
  };
};
