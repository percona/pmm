import { FC, PropsWithChildren, useMemo } from 'react';
import { NavigationContext } from './navigation.context';
import { NavItem } from 'lib/types';
import { useServiceTypes } from 'hooks/api/useServices';
import {
  addAccount,
  addAlerting,
  addDashboardItems,
  addExplore,
  addIntelligence,
} from './navigation.utils';
import { useUser } from 'contexts/user';
import { useAdvisors } from 'hooks/api/useAdvisors';
import { useColorMode } from 'hooks/theme';
import { ALL_SERVICE_TYPES } from 'lib/constants';
import { useSettings } from 'contexts/settings';
import {
  NAV_BACKUPS,
  NAV_CONFIGURATION,
  NAV_DIVIDERS,
  NAV_HELP,
  NAV_HOME_PAGE,
  NAV_INVENTORY,
  NAV_QAN,
  NAV_USERS_AND_ACCESS,
} from './navigation.constants';

export const NavigationProvider: FC<PropsWithChildren> = ({ children }) => {
  const { data: serviceTypes } = useServiceTypes();
  const { settings } = useSettings();
  const { data: advisors } = useAdvisors();
  const { colorMode, toggleColorMode } = useColorMode();
  const { user } = useUser();

  const navTree = useMemo<NavItem[]>(() => {
    const items: NavItem[] = [];
    // provide all service types for anonymous mode
    const currentServiceTypes = user
      ? serviceTypes?.serviceTypes || []
      : ALL_SERVICE_TYPES;

    items.push(NAV_HOME_PAGE);

    items.push(NAV_DIVIDERS.home);

    items.push(...addDashboardItems(currentServiceTypes, user));

    if (user && settings) {
      items.push(NAV_QAN);

      if (settings.frontend.exploreEnabled && user.isEditor) {
        items.push(addExplore(settings.frontend));
      }

      if (settings.frontend.unifiedAlertingEnabled) {
        items.push(addAlerting(settings?.alertingEnabled, user));
      }

      if (user.isEditor) {
        items.push(addIntelligence(settings, advisors || []));
      }

      if (user.isPMMAdmin) {
        items.push(NAV_DIVIDERS.inventory);

        items.push(NAV_INVENTORY);

        if (settings.backupManagementEnabled) {
          items.push(NAV_BACKUPS);
        }

        items.push(NAV_DIVIDERS.backups);

        items.push(NAV_CONFIGURATION);

        items.push(NAV_USERS_AND_ACCESS);
      }

      items.push(addAccount(user, colorMode, toggleColorMode));

      items.push(NAV_HELP);
    }

    return items;
  }, [serviceTypes, user, settings, advisors, colorMode, toggleColorMode]);

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
