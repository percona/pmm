import { getThemeById } from '@grafana/data';
import { config, getAppEvents, ThemeChangedEvent } from '@grafana/runtime';

/**
 * Changes theme to the provided one
 *
 * Based on public/app/core/services/theme.ts in Grafana
 * @param themeId
 */
export const changeTheme = async (themeId: 'light' | 'dark'): Promise<void> => {
  const oldTheme = config.theme2;

  const newTheme = getThemeById(themeId);

  // Publish Grafana ThemeChangedEvent
  getAppEvents().publish(new ThemeChangedEvent(newTheme));

  // Add css file for new theme
  if (oldTheme.colors.mode !== newTheme.colors.mode) {
    const newCssLink = document.createElement('link');
    newCssLink.rel = 'stylesheet';
    newCssLink.href = config.bootData.assets[newTheme.colors.mode];
    newCssLink.onload = () => {
      // Remove old css file after the new one has loaded to avoid flicker
      const links = document.getElementsByTagName('link');
      for (let i = 0; i < links.length; i++) {
        const link = links[i];
        if (link.href && link.href.includes(`build/grafana.${oldTheme.colors.mode}`)) {
          link.remove();
        }
      }
    };
    document.head.insertBefore(newCssLink, document.head.firstChild);
  }
};

/* ---------------------------
 * Right → left theme wiring
 * --------------------------*/

// Normalize and apply <html> attributes so CSS-based nav updates immediately
function applyHtmlTheme(modeRaw: unknown) {
  const mode: 'light' | 'dark' = String(modeRaw).toLowerCase() === 'dark' ? 'dark' : 'light';
  const html = document.documentElement;
  const scheme = mode === 'dark' ? 'percona-dark' : 'percona-light';

  if (html.getAttribute('data-theme') !== mode) {
    html.setAttribute('data-theme', mode);
  }
  if (html.getAttribute('data-md-color-scheme') !== scheme) {
    html.setAttribute('data-md-color-scheme', scheme);
  }
  (html.style as any).colorScheme = mode;

  return mode;
}

function isDevHost(host: string) {
  return host === 'localhost' || host === '127.0.0.1' || /^\d+\.\d+\.\d+\.\d+$/.test(host);
}

function resolveTargetOrigin(): string {
  try {
    const u = new URL(window.location.href);
    if (isDevHost(u.hostname)) {
      return '*';
    }
  } catch (err) {
    console.error('[pmm-compat] theme.ts not found', err);
  }
  try {
    const r = new URL(document.referrer);
    return `${r.protocol}//${r.host}`;
  } catch {
    return '*';
  }
}

const targetOrigin = resolveTargetOrigin();
let lastSentMode: 'light' | 'dark' | null = null;

// Initial apply from current Grafana theme and notify parent once
(function initThemeBridge() {
  const initial: 'light' | 'dark' = config?.theme2?.colors?.mode === 'dark' ? 'dark' : 'light';
  const mode = applyHtmlTheme(initial);
  try {
    const target = window.top && window.top !== window ? window.top : window.parent || window;
    if (lastSentMode !== mode) {
      target?.postMessage({ type: 'grafana.theme.changed', payload: { mode } }, targetOrigin);
      lastSentMode = mode;
    }
  } catch (err) {
    console.warn('[pmm-compat] failed to post initial grafana.theme.changed:', err);
  }
})();

// React to Grafana ThemeChangedEvent (Preferences change/changeTheme())
getAppEvents().subscribe(ThemeChangedEvent, (evt: any) => {
  try {
    // payload is GrafanaTheme2, not { theme: ... }
    const payload = evt?.payload;
    const next = payload?.colors?.mode ?? (payload?.isDark ? 'dark' : 'light') ?? 'light';

    const mode = applyHtmlTheme(next);

    const target = window.top && window.top !== window ? window.top : window.parent || window;
    if (lastSentMode !== mode) {
      target?.postMessage({ type: 'grafana.theme.changed', payload: { mode } }, targetOrigin);
      lastSentMode = mode;
    }
  } catch (err) {
    console.warn('[pmm-compat] Failed to handle ThemeChangedEvent/postMessage:', err);
  }
});
