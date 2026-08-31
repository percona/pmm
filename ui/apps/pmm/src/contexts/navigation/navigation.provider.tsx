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
  addSepApps,
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
import { useAuth } from 'contexts/auth';

export const NavigationProvider: FC<PropsWithChildren> = ({ children }) => {
  const { user } = useUser();
  const { isLoggedIn } = useAuth();
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

    if (user) {
      if (settings?.frontend.exploreEnabled && user.isEditor) {
        items.push(
          addExplore('grafana-metricsdrilldown-app' in settings.frontend.apps)
        );
      }

      items.push(
        addAlerting(
          settings?.alertingEnabled,
          settings?.frontend.unifiedAlertingEnabled,
          user
        )
      );

      if (user.isEditor && settings?.advisorEnabled) {
        items.push(addAdvisors(advisors || []));
      }

      items.push(NAV_DIVIDERS.inventory);

      if (user.isPMMAdmin) {
        items.push(NAV_INVENTORY);
      }

      // SEP apps mounted as native routes, grouped under "Management" and
      // placed right below Inventory so no pre-existing entry moves. Offered to
      // every signed-in user, not only admins: SEP's API serves its reads to
      // any authenticated session and holds every unsafe method to
      // administrators, so a non-admin gets a read-only view with no write
      // control rendered (PMM-15358).
      //
      // Signed-in is the rule, so anonymous is excluded: it has no Grafana
      // session cookie to exchange for a SEP bearer, and the entry would open
      // on SepAuthGate's failure card rather than on the app.
      if (!user.isAnonymous) {
        items.push(...addSepApps());
      }

      if (user.isPMMAdmin) {
        if (settings?.backupManagementEnabled) {
          items.push(NAV_BACKUPS);
        }

        items.push(NAV_DIVIDERS.backups);

        items.push(addConfiguration(status, versionInfo));

        if (settings) {
          items.push(addUsersAndAccess(settings));
        }
      }

      if (isLoggedIn) {
        items.push(addAccount(user, colorMode, toggleColorMode));
      }

      items.push(NAV_HELP);
    }

    if (!isLoggedIn) {
      items.push(NAV_SIGN_IN);
    }

    return items;
  }, [
    serviceTypes?.serviceTypes,
    user,
    haInfo,
    folders,
    settings,
    colorMode,
    toggleColorMode,
    advisors,
    status,
    versionInfo,
    isLoggedIn,
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
