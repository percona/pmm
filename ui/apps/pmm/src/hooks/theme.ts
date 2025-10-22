import { ColorModeContext } from '@percona/design';
import { useCallback, useContext, useEffect, useRef } from 'react';
import { useUpdatePreferences } from './api/useUser';
import { useGrafanaThemeSyncOnce } from './useGrafanaThemeSyncOnce';

type Mode = 'light' | 'dark';

export const useColorMode = () => {
  const { colorMode, toggleColorMode } = useContext(ColorModeContext);
  const { mutateAsync } = useUpdatePreferences();
  const colorModeRef = useRef<Mode>(colorMode);

  useEffect(() => {
    colorModeRef.current = colorMode;
  }, [colorMode]);

  useGrafanaThemeSyncOnce(colorModeRef);

  const toggleMode = useCallback(async () => {
    const prev = colorModeRef.current;
    const next: Mode = prev === 'light' ? 'dark' : 'light';

    // Optimistic UI update
    toggleColorMode();
    colorModeRef.current = next;

    try {
      await mutateAsync({ theme: next });
    } catch (err) {
      // Rollback on failure
      console.warn('[useColorMode] Persist failed, rolling back:', err);
      toggleColorMode();
      colorModeRef.current = prev;
    } finally {
      // Best-effort persistence in localStorage
      try {
        localStorage.setItem('colorMode', colorModeRef.current);
      } catch (e) {
        console.warn('[useColorMode] localStorage set failed:', e);
      }
    }
  }, [toggleColorMode, mutateAsync]);

  return { colorMode, toggleColorMode: toggleMode };
};
