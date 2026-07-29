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
    enabled: user?.isAnonymous === false && user?.isPMMAdmin,
  });
  const readonlySettings = useReadonlySettings({
    enabled: user?.isAnonymous === false && !user?.isPMMAdmin,
  });
  const frontendSettings = useFrontendSettings({
    refetchOnMount: false,
  });

  const combinedSettings = useMemo<CombinedSettings | null>(() => {
    if (user?.isAnonymous && frontendSettings.data) {
      return {
        sshKey: '',
         metricsResolutions: {
           hr: '1m',
           mr: '1m',
           lr: '1m',
         },
         dataRetention: '30d',
         pmmPublicAddress: '',
         updatesEnabled: false,
         telemetryEnabled: false,
         advisorEnabled: false,
         alertingEnabled: false,
         backupManagementEnabled: false,
         azurediscoverEnabled: false,
         enableAccessControl: false,
         frontend: frontendSettings.data,
        // check if pmm-compat-app plugin is enabled
        newUIEnabled: frontendSettings.data.apps['pmm-compat-app']?.preload,
      };
    }

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
  }, [
    user?.isAnonymous,
    user?.isPMMAdmin,
    frontendSettings.data,
    settings.data,
    readonlySettings.data,
  ]);

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
