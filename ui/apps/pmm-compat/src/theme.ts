import { getThemeById } from '@grafana/data';
import { config, getAppEvents, ThemeChangedEvent } from '@grafana/runtime';

/**
 * Changes theme to the provided one
 *
 * Based on public/app/core/services/theme.ts in Grafana
 * @param themeId
 */
console.log('[pmm-compat] compat.ts loaded');


export const changeTheme = async (themeId: 'light' | 'dark'): Promise<void> => {
  const oldTheme = config.theme2;

  console.log('Changing theme from', oldTheme, 'to', themeId);

  const newTheme = getThemeById(themeId);

  getAppEvents().publish(new ThemeChangedEvent(newTheme));

  // Add css file for new theme
  if (oldTheme.colors.mode !== newTheme.colors.mode) {
    const newCssLink = document.createElement('link');
    newCssLink.rel = 'stylesheet';
    newCssLink.href = config.bootData.assets[newTheme.colors.mode];
    newCssLink.onload = () => {
      // Remove old css file
      const bodyLinks = document.getElementsByTagName('link');
      for (let i = 0; i < bodyLinks.length; i++) {
        const link = bodyLinks[i];

        if (link.href && link.href.includes(`build/grafana.${oldTheme.colors.mode}`)) {
          // Remove existing link once the new css has loaded to avoid flickering
          // If we add new css at the same time we remove current one the page will be rendered without css
          // As the new css file is loading
          link.remove();
        }
      }
    };
    document.head.insertBefore(newCssLink, document.head.firstChild);
  }
};

getAppEvents().subscribe(ThemeChangedEvent, (evt) => {
  try {
    // payload is GrafanaTheme2, not { theme: ... }
    const payload: any = evt?.payload;
    const isDark: boolean =
      typeof payload?.isDark === 'boolean'
        ? payload.isDark
        : payload?.colors?.mode === 'dark';

    const mode: 'light' | 'dark' = isDark ? 'dark' : 'light';

    console.log('[pmm-compat] ThemeChangedEvent payload:', {
      isDark,
      mode,
      payloadMode: payload?.colors?.mode,
    });

    const parentOrigin = document.referrer ? new URL(document.referrer).origin : '*';
    console.log('[pmm-compat] Sending grafana.theme.changed → parent', { parentOrigin, mode });

    window.parent?.postMessage({ type: 'grafana.theme.changed', payload: { mode } }, parentOrigin);

    console.log('[pmm-compat] grafana.theme.changed sent');
  } catch (err) {
    console.warn('[pmm-compat] Failed to post grafana.theme.changed:', err);
  }
});

