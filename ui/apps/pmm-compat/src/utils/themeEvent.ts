import type { GrafanaTheme2 } from '@grafana/data';
import { config } from '@grafana/runtime';
import type { ColorMode } from '@pmm/shared';

/** Grafana 11+ often passes the whole theme object on the event. */
type ThemeChangedEventV11 = {
  theme?: GrafanaTheme2 | null | undefined;
};

/** Older/other shapes carry a payload with colors.mode or isDark. */
type ThemeChangedEventLegacy = {
  payload?: {
    colors?: { mode?: string | null | undefined } | null | undefined;
    isDark?: boolean | null | undefined;
  } | null | undefined;
};

/** Union that covers the known shapes across Grafana versions. */
export type ThemeChangedEventLike = ThemeChangedEventV11 | ThemeChangedEventLegacy;

/** Narrower check for GrafanaTheme2-like objects. */
function isGrafanaTheme2(value: unknown): value is GrafanaTheme2 {
  return (
    !!value &&
    typeof value === 'object' &&
    'colors' in value &&
    typeof (value as { colors?: { mode?: unknown } }).colors?.mode === 'string'
  );
}

/** Normalize arbitrary string/boolean to strict 'light' | 'dark'. */
function normalizeMode(v: unknown): ColorMode {
  return String(v).toLowerCase() === 'dark' ? 'dark' : 'light';
}

/**
 * Extract 'light' | 'dark' from various ThemeChangedEvent shapes across Grafana versions.
 * Falls back to current config.theme2.colors.mode if not present.
 */
export function parseThemeChangedEvent(evt: ThemeChangedEventLike | undefined): ColorMode {
  // 1) Newer shape: evt.theme is a GrafanaTheme2
  const themeCandidate = (evt as ThemeChangedEventV11 | undefined)?.theme;
  if (isGrafanaTheme2(themeCandidate)) {
    return normalizeMode(themeCandidate.colors.mode);
  }

  // 2) Legacy shapes: evt.payload.colors.mode OR evt.payload.isDark
  const payload = (evt as ThemeChangedEventLegacy | undefined)?.payload;
  const viaPayloadMode = payload?.colors?.mode;
  if (typeof viaPayloadMode === 'string') {
    return normalizeMode(viaPayloadMode);
  }
  if (payload?.isDark === true) {
    return 'dark';
  }

  // 3) Fallback to current runtime config
  return normalizeMode(config?.theme2?.colors?.mode);
}
