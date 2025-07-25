import {
  useFrontendSettings,
  useReadonlySettings,
  useSettings,
} from 'hooks/api/useSettings';
import { FC, PropsWithChildren, useMemo } from 'react';
import { SettingsContext } from './settings.context';
import { CombinedSettings } from './settings.context.types';
import { useUser } from 'contexts/user';

export const SettingsProvider: FC<PropsWithChildren> = ({ children }) => {
  const { user } = useUser();
  const settings = useSettings({
    enabled: !!user && user.isPMMAdmin,
  });
  const readonlySettings = useReadonlySettings({
    enabled: !!user && !user.isPMMAdmin,
  });
  const frontendSettings = useFrontendSettings({
    refetchOnMount: false,
  });
  const combinedSettings = useMemo<CombinedSettings | null>(() => {
    if (!(settings.data || readonlySettings.data) || !frontendSettings.data) {
      return null;
    }

    // admins have access to the full settings payload
    if (user?.isPMMAdmin) {
      return {
        ...settings.data!,
        frontend: frontendSettings.data!,
        // check if pmm-compat-app plugin is enabled
        newUIEnabled: frontendSettings.data.apps['pmm-compat-app']?.preload,
      };
    }

    return {
      ...readonlySettings.data!,
      frontend: frontendSettings.data!,
      // check if pmm-compat-app plugin is enabled
      newUIEnabled: frontendSettings.data.apps['pmm-compat-app']?.preload,
    };
  }, [user?.isPMMAdmin, settings, readonlySettings, frontendSettings]);

  return (
    <SettingsContext.Provider
      value={{
        isLoading:
          settings.isLoading ||
          readonlySettings.isLoading ||
          frontendSettings.isLoading,
        settings: combinedSettings,
      }}
    >
      {children}
    </SettingsContext.Provider>
  );
};
