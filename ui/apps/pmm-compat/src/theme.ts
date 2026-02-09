import { getThemeById, type GrafanaTheme2 } from '@grafana/data';
import { config, getAppEvents, ThemeChangedEvent } from '@grafana/runtime';

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
