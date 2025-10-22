// pmm-compat/src/providers/ThemeProvider.tsx
// ------------------------------------------------------
// Grafana-side theme provider + bridge:
// 1) Mirrors Grafana theme into ThemeContext (for plugins).
// 2) Applies canonical DOM attributes inside the iframe.
// 3) Notifies host UI via postMessage so host can re-theme instantly.
// ------------------------------------------------------

import React, { FC, PropsWithChildren, useEffect, useRef, useState } from 'react';
import { ThemeContext } from '@grafana/data';
import { config, getAppEvents, ThemeChangedEvent } from '@grafana/runtime';

type Mode = 'light' | 'dark';

export const ThemeProvider: FC<PropsWithChildren> = ({ children }) => {
  const [theme, setTheme] = useState(config.theme2);
  const lastSentModeRef = useRef<Mode>(config.theme2.isDark ? 'dark' : 'light');

  useEffect(() => {
    // Apply DOM attributes for the current theme on mount.
    applyIframeDomTheme(lastSentModeRef.current);
    // Also notify host once on mount (defensive; host may already be in sync).
    postModeToHost(lastSentModeRef.current);

    const sub = getAppEvents().subscribe(ThemeChangedEvent, (event) => {
      setTheme(event.payload);
      const mode: Mode = event?.payload?.isDark ? 'dark' : 'light';

      // Update iframe DOM and notify the host UI.
      applyIframeDomTheme(mode);

      // De-duplicate messages to avoid noisy bridges.
      if (lastSentModeRef.current !== mode) {
        lastSentModeRef.current = mode;
        postModeToHost(mode);
      }
    });

    // Unsubscribe on unmount (pmm-compat may be hot-reloaded in dev).
    return () => sub.unsubscribe();
  }, []);

  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
};

// ----- helpers (iframe context) -----

function applyIframeDomTheme(mode: Mode): void {
  // Update DOM attributes used by CSS variables / tokens inside Grafana iframe.
  const html = document.documentElement;
  const scheme = mode === 'dark' ? 'percona-dark' : 'percona-light';
  html.setAttribute('data-md-color-scheme', scheme);
  html.setAttribute('data-theme', mode);
  html.style.colorScheme = mode;
}

function postModeToHost(mode: Mode): void {
  // Inform the parent (host) window. We intentionally use "*" here because
  // host and iframe share origin in PMM, but in dev/proxy setups origin may differ.
  // Host side should still validate origin.
  window.parent?.postMessage({ type: 'grafana.theme.changed', payload: { mode } }, '*');
}
