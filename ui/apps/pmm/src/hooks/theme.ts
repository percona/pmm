import { ColorModeContext } from '@percona/design';
import { useContext, useEffect, useRef } from 'react';
import { useUpdatePreferences } from './api/useUser';
import { useGrafanaThemeSyncOnce } from './useGrafanaThemeSyncOnce';

type Mode = 'light' | 'dark';

export const useColorMode = () => {
  const { colorMode, toggleColorMode } = useContext(ColorModeContext);
  const { mutate } = useUpdatePreferences();

  const colorModeRef = useRef<Mode>(colorMode);
  useEffect(() => {
    colorModeRef.current = colorMode;
  }, [colorMode]);

  // stays the same, just calls the extracted hook
  useGrafanaThemeSyncOnce(colorModeRef, toggleColorMode);

  const toggleMode = () => {
    const current = colorModeRef.current;
    const next: Mode = current === 'light' ? 'dark' : 'light';

    toggleColorMode();
    colorModeRef.current = next;

    try {
      mutate({ theme: next });
    } catch (err) {
      console.warn('[useColorMode] Persist to preferences failed:', err);
    }

    try {
      localStorage.setItem('colorMode', next);
    } catch (err) {
      console.warn('[useColorMode] localStorage set failed:', err);
    }
  };

  return { colorMode, toggleColorMode: toggleMode };
};
