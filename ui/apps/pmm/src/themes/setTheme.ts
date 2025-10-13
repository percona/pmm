import { useContext, useRef } from 'react';
import { ColorModeContext } from '@percona/design';
import messenger from 'lib/messenger';
import { grafanaApi } from 'api/api';

type Mode = 'light' | 'dark';

/**
 * useSetTheme hook provides a unified way to change the theme
 * across PMM UI (left pane) and Grafana iframe (right pane),
 * keeping @percona/design, Grafana preferences, and the DOM in sync.
 */
export function useSetTheme() {
  const { colorMode, toggleColorMode } = useContext(ColorModeContext);
  const modeRef = useRef<Mode>(colorMode === 'dark' ? 'dark' : 'light');

  // Keep the reference always up-to-date with current color mode
  modeRef.current = colorMode === 'dark' ? 'dark' : 'light';

  async function setTheme(next: Mode) {
    // 1) Optimistic local update (UI reacts instantly)
    if (modeRef.current !== next) {
      toggleColorMode();
      modeRef.current = next;
    }

    // 2) Cache theme locally to avoid "white flash" before React mounts
    try {
      localStorage.setItem('colorMode', next);
    } catch (err) {
      console.warn('[useSetTheme] Failed to save theme to localStorage:', err);
    }

    // 3) Persist in Grafana user preferences (single source of truth)
    try {
      await grafanaApi.put('/user/preferences', { theme: next });
    } catch (err) {
      console.warn('[useSetTheme] Failed to persist theme to Grafana preferences:', err);
    }

    // 4) Notify the Grafana iframe (right panel)
    try {
      messenger.sendMessage({
        type: 'CHANGE_THEME',
        payload: { theme: next },
      });
    } catch (err) {
      console.warn('[useSetTheme] Failed to send CHANGE_THEME message:', err);
    }
  }

  return { setTheme };
}
