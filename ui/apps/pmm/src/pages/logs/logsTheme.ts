import { PaletteMode, Theme } from '@mui/material';
import { SeverityClass } from 'api/logs';

const SEVERITY_COLORS: Record<PaletteMode, Record<SeverityClass, string>> = {
  dark: {
    error: '#f87171',
    warning: '#fbbf24',
    info: '#60a5fa',
    other: '#71717a',
  },
  light: {
    error: '#dc2626',
    warning: '#b45309',
    info: '#2563eb',
    other: '#71717a',
  },
};

/** The colour of a severity class on the Logs page. */
export const severityColor = (cls: SeverityClass, mode: PaletteMode) =>
  SEVERITY_COLORS[mode][cls];

/** Monospace text for log lines and patterns. */
export const MONO = { typography: 'code', fontSize: 13 } as const;

/** Black surfaces for the Logs page in dark mode; the light theme is kept. */
export const logsTheme = (outer: Theme): Theme =>
  outer.palette.mode === 'light'
    ? outer
    : {
        ...outer,
        palette: {
          ...outer.palette,
          background: { default: '#000', paper: '#0d0d0f' },
          surfaces: { ...outer.palette.surfaces, low: '#000', high: '#0d0d0f' },
          divider: '#232327',
          dividers: {
            ...outer.palette.dividers,
            divider: '#232327',
            dividerStrong: '#2e2e33',
          },
          text: {
            ...outer.palette.text,
            primary: '#ececef',
            secondary: '#a1a1aa',
          },
        },
        components: {
          ...outer.components,
          MuiPaper: {
            ...outer.components?.MuiPaper,
            // MUI lightens raised dark surfaces (menus, dialogs) with a white overlay.
            styleOverrides: { root: { backgroundImage: 'none' } },
          },
        },
      };
