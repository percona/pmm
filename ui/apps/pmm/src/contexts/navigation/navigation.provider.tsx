import { type FC, type PropsWithChildren, useMemo } from 'react';
import { NavigationContext } from './navigation.context';
import type { NavItem } from 'types/navigation.types';
import { useServiceTypes } from 'hooks/api/useServices';
import { useFolders } from 'hooks/api/useFolders';
import { useUser } from 'contexts/user';
import { useColorMode } from 'hooks/theme';
import { INTERVALS_MS } from 'lib/constants';
import { useSettings } from 'contexts/settings';
import { useUpdates } from 'contexts/updates';
import { useLocalStorage } from 'hooks/utils/useLocalStorage';
import { useHaInfo } from 'hooks/api/useHA';
import { useAuth } from 'contexts/auth';
import { ServiceType } from 'types/services.types';
import { buildNavTree } from './navigation.tree';

export const NavigationProvider: FC<PropsWithChildren> = ({ children }) => {
  const { user } = useUser();
  const { isLoggedIn, isAnonymous } = useAuth();
  const { data: serviceTypes } = useServiceTypes({
    enabled: !!user,
    refetchInterval: INTERVALS_MS.SERVICE_TYPES,
  });
  const { settings } = useSettings();
  const { data: folders = [] } = useFolders();
  const { colorMode, toggleColorMode } = useColorMode();
  const { status, versionInfo } = useUpdates();
  const [navOpen, setNavOpen] = useLocalStorage<boolean>(
    'pmm-ui.sidebar.expanded',
    true
  );
  const { data: haInfo } = useHaInfo({
    enabled: !!user,
  });

  const navTree = useMemo<NavItem[]>(
    () =>
      buildNavTree({
        user,
        isLoggedIn,
        haInfo,
        settings: settings ?? undefined,
        // Anonymous visitors can't list service types, so they get every
        // database's dashboards; otherwise empty while the types are loading.
        serviceTypes: isAnonymous
          ? Object.values(ServiceType)
          : serviceTypes?.serviceTypes || [],
        folders,
        updateStatus: status,
        versionInfo,
        colorMode,
        toggleColorMode,
      }),
    [
      user,
      isLoggedIn,
      isAnonymous,
      haInfo,
      settings,
      serviceTypes?.serviceTypes,
      folders,
      status,
      versionInfo,
      colorMode,
      toggleColorMode,
    ]
  );

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
