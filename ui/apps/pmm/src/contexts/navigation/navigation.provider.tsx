import { FC, PropsWithChildren, useCallback, useMemo } from 'react';
import { NavigationContext } from './navigation.context';
import { NavItem } from 'types/navigation.types';
import { useUser } from 'contexts/user';
import { useColorMode } from 'hooks/theme';
import { useSettings } from 'contexts/settings';
import { useUpdates } from 'contexts/updates';
import { useLocalStorage } from 'hooks/utils/useLocalStorage';
import { useHaInfo } from 'hooks/api/useHA';
import { useStarredDashboards } from 'hooks/api/useStarredDashboards';
import { useAuth } from 'contexts/auth';
import {
  buildProposedNavTree,
  DEFAULT_STARRED_APPS,
} from './navigation.proposal';

export const STARRED_APPS_KEY = 'pmm-ui.sidebar.starred-apps';

export const NavigationProvider: FC<PropsWithChildren> = ({ children }) => {
  const { user } = useUser();
  const { isLoggedIn } = useAuth();
  const { settings } = useSettings();
  const { colorMode, toggleColorMode } = useColorMode();
  const { status, versionInfo } = useUpdates();
  const [navOpen, setNavOpen] = useLocalStorage<boolean>(
    'pmm-ui.sidebar.expanded',
    true
  );
  const { data: haInfo } = useHaInfo({
    enabled: user?.isAnonymous === false,
  });
  const { data: starredDashboards } = useStarredDashboards({
    enabled: user?.isAnonymous === false,
  });
  const [starredApps, setStarredApps] = useLocalStorage<string[]>(
    STARRED_APPS_KEY,
    DEFAULT_STARRED_APPS
  );
  const toggleStarredApp = useCallback(
    (id: string) =>
      setStarredApps(
        starredApps.includes(id)
          ? starredApps.filter((starred) => starred !== id)
          : [...starredApps, id]
      ),
    [starredApps, setStarredApps]
  );

  const navTree = useMemo<NavItem[]>(
    () =>
      buildProposedNavTree({
        user,
        isLoggedIn,
        haInfo,
        settings: settings ?? undefined,
        updateStatus: status,
        versionInfo,
        colorMode,
        toggleColorMode,
        starredDashboards,
        starredApps,
      }),
    [
      user,
      isLoggedIn,
      haInfo,
      settings,
      status,
      versionInfo,
      colorMode,
      toggleColorMode,
      starredDashboards,
      starredApps,
    ]
  );

  return (
    <NavigationContext.Provider
      value={{
        navTree,
        navOpen,
        setNavOpen,
        starredApps,
        toggleStarredApp,
      }}
    >
      {children}
    </NavigationContext.Provider>
  );
};
