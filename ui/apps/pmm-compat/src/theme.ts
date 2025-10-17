import { getThemeById } from '@grafana/data';
import { config, getAppEvents, ThemeChangedEvent } from '@grafana/runtime';

/**
 * Changes theme to the provided one.
 * Based on public/app/core/services/theme.ts in Grafana
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
  (html.style as CSSStyleDeclaration).colorScheme = mode;

  return mode;
}

const isIp = (h: string) => /^\d+\.\d+\.\d+\.\d+$/.test(h);
const isDevHost = (h: string) => h === 'localhost' || h === '127.0.0.1' || isIp(h);

const parseOrigin = (u: string | URL | null | undefined): string | null => {
  try {
    const url = typeof u === 'string' ? new URL(u) : u instanceof URL ? u : null;
    return url ? `${url.protocol}//${url.host}` : null;
  } catch {
    return null;
  }
};

/**
 * Resolve initial target origin (may be '*' in dev).
 * - Dev: start with '*' to support split hosts/ports (vite + docker).
 * - Prod: concrete origin (document.referrer → window.location.origin).
 */
function resolveInitialTargetOrigin(): string {
  const loc = new URL(window.location.href);
  if (isDevHost(loc.hostname)) {
    return '*';
  }
  const ref = parseOrigin(document.referrer);
  return ref ?? `${loc.protocol}//${loc.host}`;
}

/** Safely obtain a Window to post to (top if cross-framed, otherwise parent/self). */
function resolveTargetWindow(): Window | null {
  try {
    if (window.top && window.top !== window) {
      return window.top;
    }
    if (window.parent) {
      return window.parent;
    }
  } catch (err) {
    console.warn('[pmm-compat] Failed to send handshake:', err);
  }
  return window;
}

/** Runtime-locked origin (handshake will tighten '*' in dev). */
const targetOriginRef = { current: resolveInitialTargetOrigin() };
let lastSentMode: 'light' | 'dark' | null = null;

/** Send helper that always uses the current locked origin. */
function sendToParent(msg: unknown) {
  const w = resolveTargetWindow();
  if (!w) {
    return;
  }
  w.postMessage(msg, targetOriginRef.current);
}

/** Dev-only handshake: lock '*' to the real origin after the first ACK. */
(function setupOriginHandshake() {
  const isDev = isDevHost(new URL(window.location.href).hostname);
  if (!isDev || targetOriginRef.current !== '*') {
    return;
  }

  // Ask parent for its origin once
  try {
    sendToParent({ type: 'pmm.handshake' });
  } catch {
    // ignore
  }

  const onAck = (e: MessageEvent<{ type?: string }>) => {
    if (e?.data?.type !== 'pmm.handshake.ack') {
      return;
    }
    // Lock to the explicit origin provided by the parent
    targetOriginRef.current = e.origin || targetOriginRef.current;
    window.removeEventListener('message', onAck);
  };
  window.addEventListener('message', onAck);
})();

// Initial apply from current Grafana theme and notify parent once
(function initThemeBridge() {
  const initial: 'light' | 'dark' = config?.theme2?.colors?.mode === 'dark' ? 'dark' : 'light';
  const mode = applyHtmlTheme(initial);
  try {
    if (lastSentMode !== mode) {
      sendToParent({ type: 'grafana.theme.changed', payload: { mode } });
      lastSentMode = mode;
    }
  } catch (err) {
    console.warn('[pmm-compat] failed to post initial grafana.theme.changed:', err);
  }
})();

// React to Grafana ThemeChangedEvent (Preferences change/changeTheme())
getAppEvents().subscribe(ThemeChangedEvent, (evt: unknown) => {
  try {
    // Type guard for expected payload structure
    if (typeof evt === 'object' && evt !== null && 'payload' in evt) {
      const payload = (evt as { payload?: unknown }).payload;
      const next =
        typeof payload === 'object' && payload !== null && 'colors' in payload
          ? (payload as { colors?: { mode?: string }; isDark?: boolean }).colors?.mode ??
            ((payload as { isDark?: boolean }).isDark ? 'dark' : 'light') ??
            'light'
          : 'light';

      const mode = applyHtmlTheme(next);

      if (lastSentMode !== mode) {
        sendToParent({ type: 'grafana.theme.changed', payload: { mode } });
        lastSentMode = mode;
      }
    }
  } catch (err) {
    console.warn('[pmm-compat] Failed to handle ThemeChangedEvent/postMessage:', err);
  }
});
