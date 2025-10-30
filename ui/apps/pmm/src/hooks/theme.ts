import { ColorModeContext } from '@percona/design';
import { useContext, useRef } from 'react';
import { useUpdatePreferences } from './api/useUser';
import messenger from 'lib/messenger';
import { ColorMode, MessageType } from '@pmm/shared';

function normalizeMode(v: unknown): ColorMode {
  if (typeof v === 'string' && v.toLowerCase() === 'dark') return 'dark';
  if (v === true) return 'dark';
  return 'light';
}

export const useColorMode = () => {
  const { colorMode, toggleColorMode } = useContext(ColorModeContext);
  const { mutate,  mutateAsync: updatePreferences  } = useUpdatePreferences();
  const modeRef = useRef<ColorMode>(normalizeMode(colorMode));

  // Keep the reference always up-to-date with current color mode
  modeRef.current = normalizeMode(colorMode);

  const onToggle = () => {
    const next = colorMode === 'light' ? 'dark' : 'light';

    // 1) local apply (left UI)
    toggleColorMode();

    // 2) tell Grafana iframe to switch immediately
    messenger.sendMessage({
      type: 'CHANGE_THEME' as MessageType,
      payload: { theme: next },
    });

    // 3) persist in Grafana Preferences
    mutate({ theme: next });
  };

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
      try {
        await updatePreferences({ theme: next });
      } catch (err) {
        console.warn('[useColorMode] Failed to persist theme:', err);
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

  async function setFromGrafana(next: ColorMode | string | boolean) {
    await setThemeBase(next, { broadcast: false, persist: false });
  }

  return { colorMode, toggleColorMode: onToggle, setFromGrafana };
};
