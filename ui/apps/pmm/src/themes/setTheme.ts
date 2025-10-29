import { useContext, useRef } from 'react';
import { ColorModeContext } from '@percona/design';
import messenger from 'lib/messenger';
import { ColorMode } from '@pmm/shared';
import { grafanaApi } from 'api/api';
import { useUpdatePreferences } from '../hooks/api/useUser';

/** Normalize any incoming value to 'light' | 'dark'. */
function normalizeMode(v: unknown): ColorMode {
  if (typeof v === 'string' && v.toLowerCase() === 'dark') return 'dark';
  if (v === true) return 'dark';
  return 'light';
}

/**
 * useSetTheme centralizes theme changes for:
 * - local PMM UI (left)
 * - persistence to Grafana preferences
 * - broadcasting to Grafana iframe (right)
 */
export function useSetTheme() {
  const { colorMode, toggleColorMode } = useContext(ColorModeContext);
  const modeRef = useRef<ColorMode>(normalizeMode(colorMode));

  const { mutateAsync: updatePreferences } = useUpdatePreferences();

  // Keep the reference always up-to-date with current color mode
  modeRef.current = normalizeMode(colorMode);

  /** Apply theme locally via design system only (no direct DOM mutations). */
  const applyLocal = (nextRaw: unknown) => {
    const next = normalizeMode(nextRaw);
    if (modeRef.current !== next) {
      // design system exposes only toggle, so flip when needed
      toggleColorMode();
      modeRef.current = next;
    }
  };

  /** Low-level primitive to apply/persist/broadcast theme as needed. */
  const setThemeBase = async (
    nextRaw: unknown,
    opts: { broadcast?: boolean; persist?: boolean } = {
      broadcast: true,
      persist: true,
    }
  ) => {
    const next = normalizeMode(nextRaw);

    // 1) Local apply (instant, idempotent)
    applyLocal(next);

    // 2) Persist to Grafana (only for left-initiated actions)
    if (opts.persist) {
      await updatePreferences({ theme: next });
      try {
        await grafanaApi.put('/user/preferences', { theme: next });
      } catch (err) {
        console.warn(
          '[useSetTheme] Failed to persist theme to Grafana preferences:',
          err
        );
      }
    }

    // 3) Notify iframe (only when we are the initiator, not when syncing from Grafana)
    if (opts.broadcast) {
      try {
        messenger.sendMessage({
          type: 'CHANGE_THEME',
          payload: { mode: next },
        });
      } catch (err) {
        console.warn('[useSetTheme] Failed to send CHANGE_THEME message:', err);
      }
    }
  };

  /**
   * Public API:
   * - setTheme(next): left action — apply + persist + broadcast.
   * - setFromGrafana(next): right→left sync — apply only (no persist, no broadcast).
   */
  async function setTheme(next: ColorMode | string | boolean) {
    await setThemeBase(next, { broadcast: true, persist: true });
  }

  async function setFromGrafana(next: ColorMode | string | boolean) {
    await setThemeBase(next, { broadcast: false, persist: false });
  }

  return { setTheme, setFromGrafana };
}
