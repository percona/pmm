import type { CSSObject, Theme } from '@mui/material/styles';
import { themes } from 'prism-react-renderer';
import type { PrismTheme } from 'prism-react-renderer';

// Background and text color always come from the prism scheme (applied via the inline `style`),
// so every block — highlighted or plain — shares the picked scheme's canvas.
export const preSx =
  (wrap: boolean) =>
  (theme: Theme): CSSObject => ({
    margin: 0,
    ...theme.typography.code,
    fontSize: '0.875rem',
    lineHeight: 1.5,
    border: `1px solid ${theme.palette.dividers?.contour ?? theme.palette.divider}`,
    borderRadius: '5px',
    padding: theme.spacing(1.25, 1.5),
    ...(wrap ? { whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' } : { overflowX: 'auto' }),
  });

// Scheme follows the app color mode — locked to brand-aligned light/dark for consistency.
export const resolveScheme = (mode: Theme['palette']['mode']): PrismTheme =>
  mode === 'dark' ? themes.okaidia : themes.nightOwlLight;
