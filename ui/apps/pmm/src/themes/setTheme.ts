import { useContext, useRef } from 'react';
import { ColorModeContext } from '@percona/design';
import messenger from 'lib/messenger';
import { grafanaApi } from 'api/api';

type Mode = 'light' | 'dark';

/** Normalizes any incoming value to 'light' | 'dark'. */
function normalizeMode(v: unknown): Mode {
  if (typeof v === 'string' && v.toLowerCase() === 'dark') return 'dark';
  if (v === true) return 'dark';
  return 'light';
}

/** Idempotently applies theme attributes to <html>. */
function applyDocumentTheme(mode: Mode) {
  const html = document.documentElement as HTMLElement & {
    style: CSSStyleDeclaration & { colorScheme?: string };
  };
  const scheme = mode === 'dark' ? 'percona-dark' : 'percona-light';
  const wantDark = mode === 'dark';
  const hasDarkClass = html.classList.contains('dark');

  // If everything (including the Tailwind 'dark' class) is already correct, skip.
  if (
    html.getAttribute('data-theme') === mode &&
    html.getAttribute('data-md-color-scheme') === scheme &&
    html.style.colorScheme === mode &&
    hasDarkClass === wantDark
  ) {
    return;
  }

  html.classList.toggle('dark', wantDark);

  html.setAttribute('data-theme', mode);
  html.setAttribute('data-md-color-scheme', scheme);
  html.style.colorScheme = mode;
}

/**
 * useSetTheme centralizes theme changes for:
 * - local PMM UI (left)
 * - persistence to Grafana preferences
 * - broadcasting to Grafana iframe (right)
 */
export function useSetTheme() {
  const { colorMode, toggleColorMode } = useContext(ColorModeContext);
  const modeRef = useRef<Mode>(normalizeMode(colorMode));

  // Keep the reference always up-to-date with current color mode
  modeRef.current = normalizeMode(colorMode);

  /** Apply theme locally (left UI) in an idempotent way */
  const applyLocal = (nextRaw: unknown) => {
    const next = normalizeMode(nextRaw);
    if (modeRef.current !== next) {
      // design system exposes only toggle, so flip when needed
      toggleColorMode();
      modeRef.current = next;
    }
    // Ensure <html> attributes match immediately (CSS consumes these)
    applyDocumentTheme(next);

    try {
      localStorage.setItem('colorMode', next);
    } catch (err) {
      // eslint-disable-next-line no-console
      console.warn('[useSetTheme] Failed to save theme to localStorage:', err);
    }
  };

  /** Low-level primitive with options to avoid ping-pong and over-persisting */
  const setThemeBase = async (
    nextRaw: unknown,
    opts: { broadcast?: boolean; persist?: boolean } = {
      broadcast: true,
      persist: true,
    }
  ) => {
    const next = normalizeMode(nextRaw);

    // 1) local apply (instant, idempotent)
    applyLocal(next);

    // 2) persist to Grafana (only for left-initiated actions)
    if (opts.persist) {
      try {
        await grafanaApi.put('/user/preferences', { theme: next });
      } catch (err) {
        // eslint-disable-next-line no-console
        console.warn(
          '[useSetTheme] Failed to persist theme to Grafana preferences:',
          err
        );
      }
    }

    // 3) notify iframe (only when we are the initiator, not when we sync from Grafana)
    if (opts.broadcast) {
      try {
        messenger.sendMessage({
          type: 'CHANGE_THEME',
          payload: { theme: next },
        });
      } catch (err) {
        // eslint-disable-next-line no-console
        console.warn('[useSetTheme] Failed to send CHANGE_THEME message:', err);
      }
    }
  };

  /**
   * Public API kept backward compatible:
   * - setTheme(next): left action — apply + persist + broadcast (same behavior as before)
   * - setFromGrafana(next): right→left sync — apply only (no persist, no broadcast)
   */
  async function setTheme(next: Mode | string | boolean) {
    await setThemeBase(next, { broadcast: true, persist: true });
  }

  async function setFromGrafana(next: Mode | string | boolean) {
    await setThemeBase(next, { broadcast: false, persist: false });
  }

  return { setTheme, setFromGrafana };
}
