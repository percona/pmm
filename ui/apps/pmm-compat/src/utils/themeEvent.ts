import type { GrafanaTheme2 } from '@grafana/data';
import { config } from '@grafana/runtime';

export type ThemeValue = 'light' | 'dark';

/**
 * Extract 'light' | 'dark' from various ThemeChangedEvent shapes across Grafana versions.
 * Falls back to current config.theme2.colors.mode if not present.
 */
export function parseThemeChangedEvent(evt: unknown): ThemeValue {
  try {
    // Newer shape: evt.theme: GrafanaTheme2
    const themeObj = (evt as any)?.theme as GrafanaTheme2 | undefined;
    const viaTheme = themeObj?.colors?.mode;
    if (viaTheme === 'dark') {
      return 'dark';
    }
    if (viaTheme === 'light') {
      return 'light';
    }

    // Older shapes: evt.payload.colors.mode OR evt.payload.isDark
    const payload = (evt as any)?.payload;
    const viaPayloadMode = payload?.colors?.mode;
    if (viaPayloadMode === 'dark') {
      return 'dark';
    }
    if (viaPayloadMode === 'light') {
      return 'light';
    }
    if (payload?.isDark === true) {
      return 'dark';
    }
  } catch {
    // ignore parsing errors and use fallback
  }

  return config?.theme2?.colors?.mode === 'dark' ? 'dark' : 'light';
}
