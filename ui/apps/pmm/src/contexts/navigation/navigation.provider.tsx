import { FC, PropsWithChildren, useMemo } from 'react';
import { NavigationContext } from './navigation.context';
import { NavItem } from 'types/navigation.types';
import { useUser } from 'contexts/user';
import { useColorMode } from 'hooks/theme';
import { useSettings } from 'contexts/settings';
import { useUpdates } from 'contexts/updates';
import { useLocalStorage } from 'hooks/utils/useLocalStorage';
import { useHaInfo } from 'hooks/api/useHA';
import { useAuth } from 'contexts/auth';
import { buildProposedNavTree } from './navigation.proposal';

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
