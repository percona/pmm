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
