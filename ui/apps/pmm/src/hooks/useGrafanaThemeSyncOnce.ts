import { useEffect, useRef } from 'react';
import { useSetTheme } from 'themes/setTheme';

type Mode = 'light' | 'dark';

interface GrafanaPreferences {
  theme?: 'dark' | 'light' | string;
}

declare global {
  interface Window {
    __pmm_has_theme_listener__?: string;
  }
}

/**
 * One-time sync with Grafana preferences and live postMessage events.
 * Accepts messages from any origin in dev to support mixed origins.
 */
export function useGrafanaThemeSyncOnce(
  colorModeRef: React.MutableRefObject<Mode>,
  _toggleColorMode: () => void
) {
  const syncedRef = useRef(false);
  const { setFromGrafana } = useSetTheme();

  useEffect(() => {
    if (syncedRef.current) return;
    syncedRef.current = true;

    const ensureMode = (desired: Mode) => {
      setFromGrafana(desired).catch((err) => {
        console.warn('[useGrafanaThemeSyncOnce] apply failed:', err);
      });
      colorModeRef.current = desired;
      try {
        localStorage.setItem('colorMode', desired);
      } catch (err) {
        console.warn('[useGrafanaThemeSyncOnce] localStorage set failed:', err);
      }
    };

    fetch('/graph/api/user/preferences', { credentials: 'include' })
      .then(async (r): Promise<GrafanaPreferences | null> => {
        return r.ok ? ((await r.json()) as GrafanaPreferences) : null;
      })
      .then((prefs) => {
        if (!prefs) return;
        const desired: Mode = prefs.theme === 'dark' ? 'dark' : 'light';
        ensureMode(desired);
      })
      .catch((err) => {
        console.warn('[useGrafanaThemeSyncOnce] read prefs failed:', err);
      });

    const onMsg = (
      e: MessageEvent<{
        type?: string;
        payload?: { mode?: string; payloadMode?: string; isDark?: boolean };
      }>
    ) => {
      const data = e.data;
      if (!data || data.type !== 'grafana.theme.changed') return;
      const p = data.payload ?? {};
      const raw = p.mode ?? p.payloadMode ?? (p.isDark ? 'dark' : 'light');
      const desired: Mode =
        String(raw).toLowerCase() === 'dark' ? 'dark' : 'light';
      ensureMode(desired);
    };

    window.addEventListener('message', onMsg);
    return () => window.removeEventListener('message', onMsg);
  }, [colorModeRef, setFromGrafana, _toggleColorMode]);
}
