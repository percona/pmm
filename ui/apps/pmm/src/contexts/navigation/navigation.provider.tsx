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
  addHighAvailability,
  addUsersAndAccess,
  addHomePage,
} from './navigation.utils';
import { useUser } from 'contexts/user';
import { useAdvisors } from 'hooks/api/useAdvisors';
import { useColorMode } from 'hooks/theme';
import { INTERVALS_MS } from 'lib/constants';
import { useSettings } from 'contexts/settings';
import {
  NAV_BACKUPS,
  NAV_DIVIDERS,
  NAV_HELP,
  NAV_INVENTORY,
  NAV_QAN,
  NAV_SIGN_IN,
} from './navigation.constants';
import { useFolders } from 'hooks/api/useFolders';
import { useUpdates } from 'contexts/updates';
import { useLocalStorage } from 'hooks/utils/useLocalStorage';
import { useHaInfo } from 'hooks/api/useHA';

export const NavigationProvider: FC<PropsWithChildren> = ({ children }) => {
  const { user } = useUser();
  const { data: serviceTypes } = useServiceTypes({
    enabled: !!user,
    refetchInterval: INTERVALS_MS.SERVICE_TYPES,
  });
  const { settings } = useSettings();
  const { data: advisors } = useAdvisors({
    enabled: !!user?.isEditor,
  });
  const { data: folders = [] } = useFolders();
  const { colorMode, toggleColorMode } = useColorMode();
  const { status, versionInfo } = useUpdates();
  const [navOpen, setNavOpen] = useLocalStorage<boolean>(
    'pmm-ui.sidebar.expanded',
    true
  );
  const { data: haInfo } = useHaInfo({
    enabled: user?.isAnonymous === false,
  });

  const navTree = useMemo<NavItem[]>(() => {
    const items: NavItem[] = [];
    // use fetched service types, falling back to an empty list while unavailable
    const currentServiceTypes = serviceTypes?.serviceTypes || [];

    items.push(addHomePage(user?.preferences));

    if (haInfo.enabled) {
      items.push(addHighAvailability(haInfo));
    }

    items.push(NAV_DIVIDERS.home);

    items.push(...addDashboardItems(currentServiceTypes, folders, user));

    items.push(NAV_QAN);

    if (user && settings) {
      if (settings.frontend.exploreEnabled && user.isEditor) {
        items.push(
          addExplore('grafana-metricsdrilldown-app' in settings.frontend.apps)
        );
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

        items.push(addUsersAndAccess(settings));
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
    haInfo,
    toggleColorMode,
  ]);

  return (
    <NavigationContext.Provider
      value={{
        navTree,
        navOpen,
        setNavOpen,
      }}
    >
      {children}
    </NavigationContext.Provider>
  );
};
