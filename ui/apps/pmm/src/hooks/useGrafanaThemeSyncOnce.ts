import { useEffect, useRef } from 'react';

type Mode = 'light' | 'dark';

/**
 * One-time sync with Grafana preferences and live postMessage events.
 * It ensures @percona/design color mode matches Grafana (iframe).
 * NOTE: We accept messages from any origin to support dev with mixed origins.
 */
export function useGrafanaThemeSyncOnce(
  colorModeRef: React.MutableRefObject<Mode>,
  toggleColorMode: () => void
) {
  const syncedRef = useRef(false);

  useEffect(() => {
    if (syncedRef.current) return;
    syncedRef.current = true;

    // Idempotent setter implemented via toggle-only API.
    const ensureMode = (desired: Mode) => {
      const current = colorModeRef.current;
      // eslint-disable-next-line no-console
      if (desired !== current) {
        try {
          toggleColorMode();
          colorModeRef.current = desired;
          try {
            localStorage.setItem('colorMode', desired);
          } catch (err) {
            // eslint-disable-next-line no-console
            console.warn('[useGrafanaThemeSyncOnce] localStorage set failed:', err);
          }
          // eslint-disable-next-line no-console
          console.log('[useGrafanaThemeSyncOnce] applied mode', desired);
        } catch (err) {
          // eslint-disable-next-line no-console
          console.warn('[useGrafanaThemeSyncOnce] apply failed:', err);
        }
      }
    };

    // Initial sync from Grafana user preferences
    fetch('/graph/api/user/preferences', { credentials: 'include' })
      .then((r) => {
        // eslint-disable-next-line no-console
        console.log('[useGrafanaThemeSyncOnce] prefs response ok?', r.ok);
        return r.ok ? r.json() : null;
      })
      .then((prefs) => {
        if (!prefs) return;
        const desired: Mode = prefs?.theme === 'dark' ? 'dark' : 'light';
        // eslint-disable-next-line no-console
        console.log('[useGrafanaThemeSyncOnce] initial desired from prefs:', desired);
        ensureMode(desired);
      })
      .catch((err) => {
        // eslint-disable-next-line no-console
        console.warn('[useGrafanaThemeSyncOnce] read prefs failed:', err);
      });

    // Live sync: listen to Grafana → PMM theme messages
    const onMsg = (e: MessageEvent) => {
      // Debug every message to see origin + payload in dev/prod
      // eslint-disable-next-line no-console
      console.log('[useGrafanaThemeSyncOnce] window.message', { origin: e.origin, data: e.data });

      // Accept in dev/prod; guard only by message type (no strict origin guard)
      if (!e?.data || e.data.type !== 'grafana.theme.changed') return;

      const desired: Mode = e.data?.payload?.mode === 'dark' ? 'dark' : 'light';
      // eslint-disable-next-line no-console
      console.log('[useGrafanaThemeSyncOnce] grafana.theme.changed → desired:', desired);

      ensureMode(desired);
    };

    window.addEventListener('message', onMsg);
    return () => window.removeEventListener('message', onMsg);
  }, [colorModeRef, toggleColorMode]);
}
