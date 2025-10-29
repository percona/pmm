import { getThemeById, type GrafanaTheme2 } from '@grafana/data';
import { config, getAppEvents, ThemeChangedEvent } from '@grafana/runtime';
import type { CrossFrameMessenger, Message } from '@pmm/shared';

/**
 * Normalize any input to strict 'light' | 'dark'.
 */
const normalizeMode = (incoming: unknown): 'light' | 'dark' => {
  return String(incoming).toLowerCase() === 'dark' ? 'dark' : 'light';
};

/**
 * Apply Grafana theme by id and ensure the proper CSS bundle is loaded.
 * Based on Grafana's public/app/core/services/theme.ts (trimmed).
 */
const applyGrafanaTheme = async (mode: 'light' | 'dark'): Promise<GrafanaTheme2> => {
  const oldTheme = config.theme2;
  const newTheme = getThemeById(mode);

  // Publish Grafana ThemeChangedEvent so Grafana UI re-themes itself
  getAppEvents().publish(new ThemeChangedEvent(newTheme));

  // If mode actually changed, ensure the correct CSS bundle is present
  if (oldTheme.colors.mode !== newTheme.colors.mode) {
    const cssHref = config.bootData.assets[newTheme.colors.mode];
    if (cssHref) {
      const newCssLink = document.createElement('link');
      newCssLink.rel = 'stylesheet';
      newCssLink.href = cssHref;
      newCssLink.onload = () => {
        // Remove the opposite mode's stylesheet once the new one is safely loaded
        const links = Array.from(document.querySelectorAll('link[rel="stylesheet"]')) as HTMLLinkElement[];
        for (const link of links) {
          if (link !== newCssLink && typeof link.href === 'string') {
            const isOldDark = oldTheme.colors.mode === 'dark' && link.href.includes('/dark.');
            const isOldLight = oldTheme.colors.mode === 'light' && link.href.includes('/light.');
            if (isOldDark || isOldLight) {
              link.parentElement?.removeChild(link);
            }
          }
        }
      };
      document.head.appendChild(newCssLink);
    }
  }

  return newTheme;
};

/**
 * Public API kept for callers inside this plugin (no HTML attributes here).
 */
export const changeTheme = async (themeId: 'light' | 'dark'): Promise<void> => {
  await applyGrafanaTheme(themeId);
};

/**
 * Initialize theme sync inside the Grafana iframe.
 * - Single subscription to Grafana ThemeChangedEvent => notify PMM UI (left).
 * - Listen to CHANGE_THEME from PMM UI => apply locally via changeTheme().
 * - Perform initial one-shot sync after listeners are in place.
 * - No IIFEs, no window.postMessage, no origin handshake, no HTML attributes.
 */
export const initialize = (messenger: CrossFrameMessenger): void => {
  // Guard to avoid double init if initialize() gets called twice
  if ((window as any).__pmmThemeInitDone) {
    return;
  }
  (window as any).__pmmThemeInitDone = true;

  // Outgoing: when Grafana emits ThemeChangedEvent, tell PMM UI once per change
  const onThemeChanged = (evt: ThemeChangedEvent) => {
    // In Grafana 10+, the new theme is carried in the event's theme
    const nextMode = normalizeMode((evt as any)?.theme?.colors?.mode ?? config.theme2.colors.mode);
    messenger.sendMessage({
      type: 'GRAFANA_THEME_CHANGED',
      payload: { mode: nextMode },
    });
  };

  // Subscribe once; PMM side should avoid ping-pong with its own guard flag
  getAppEvents().subscribe(ThemeChangedEvent, onThemeChanged);

  // Incoming: apply theme when PMM UI asks us to change
  messenger.addListener<'CHANGE_THEME', { mode?: string }>({
    type: 'CHANGE_THEME',
    onMessage: async (msg: Message<'CHANGE_THEME', { mode?: string }>) => {
      const requested = normalizeMode(msg.payload?.mode);
      await changeTheme(requested);
    },
  });

  // Initial one-shot sync (after listeners are registered)
  const currentMode = normalizeMode(config.theme2.colors.mode);
  messenger.sendMessage({
    type: 'GRAFANA_THEME_CHANGED',
    payload: { mode: currentMode },
  });
};
