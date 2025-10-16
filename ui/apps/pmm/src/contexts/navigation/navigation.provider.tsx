import { FC, PropsWithChildren, useMemo } from 'react';
import { NavigationContext } from './navigation.context';
import { NavItem } from 'types/navigation.types';
import { useServiceTypes } from 'hooks/api/useServices';
import {
  addAccount,
  addAdvisors,
  addAlerting,
  addConfiguration,
  addDashboardItems,
  addExplore,
} from './navigation.utils';
import { useUser } from 'contexts/user';
import { useAdvisors } from 'hooks/api/useAdvisors';
import { useColorMode } from 'hooks/theme';
import { ALL_SERVICE_TYPES, INTERVALS_MS } from 'lib/constants';
import { useSettings } from 'contexts/settings';
import {
  NAV_BACKUPS,
  NAV_DIVIDERS,
  NAV_HELP,
  NAV_HOME_PAGE,
  NAV_INVENTORY,
  NAV_QAN,
  NAV_SIGN_IN,
} from './navigation.constants';
import { useFolders } from 'hooks/api/useFolders';
import { useUpdates } from 'contexts/updates';

export const NavigationProvider: FC<PropsWithChildren> = ({ children }) => {
  const { data: serviceTypes } = useServiceTypes({
    refetchInterval: INTERVALS_MS.SERVICE_TYPES,
  });
  const { settings } = useSettings();
  const { data: advisors } = useAdvisors();
  const { data: folders = [] } = useFolders();
  const { colorMode, toggleColorMode } = useColorMode();
  const { user } = useUser();
  const { status, versionInfo } = useUpdates();

  const navTree = useMemo<NavItem[]>(() => {
    const items: NavItem[] = [];
    // provide all service types for anonymous mode
    const currentServiceTypes = user
      ? serviceTypes?.serviceTypes || []
      : ALL_SERVICE_TYPES;

    items.push(NAV_HOME_PAGE);

    items.push(NAV_DIVIDERS.home);

    items.push(...addDashboardItems(currentServiceTypes, folders, user));

    if (user && settings) {
      items.push(NAV_QAN);

      if (settings.frontend.exploreEnabled && user.isEditor) {
        items.push(addExplore(settings.frontend));
      }

      if (settings.frontend.unifiedAlertingEnabled) {
        items.push(addAlerting(settings?.alertingEnabled, user));
      }

      if (user.isEditor && settings.advisorEnabled) {
        items.push(addAdvisors(advisors || []));
      }

      if (user.isPMMAdmin) {
        items.push(NAV_DIVIDERS.inventory);

        items.push(NAV_INVENTORY);

        if (settings.backupManagementEnabled) {
          items.push(NAV_BACKUPS);
        }

        items.push(NAV_DIVIDERS.backups);

        items.push(addConfiguration(status, versionInfo));
      }

      items.push(addAccount(user, colorMode, toggleColorMode));

      items.push(NAV_HELP);
    } else {
      items.push(NAV_SIGN_IN);
    }

    return items;
  }, [
    status,
    versionInfo,
    serviceTypes,
    folders,
    user,
    settings,
    advisors,
    colorMode,
    toggleColorMode,
  ]);

  return (
    <NavigationContext.Provider
      value={{
        navTree,
      }}
    >
      {children}
    </NavigationContext.Provider>
  );
};
