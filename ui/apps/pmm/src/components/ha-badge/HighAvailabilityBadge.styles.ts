import { Theme } from '@mui/material/styles';
import { PEAK_DARK_THEME, PEAK_LIGHT_THEME } from '@pmm/shared';

export const getStyles = (theme: Theme) => ({
  healthy: {
    color: theme.palette.text.primary,
    borderColor: theme.palette.text.primary,
  },
  degraded: {
    color:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.brand.sunrise[700]
        : PEAK_DARK_THEME.extra.yellow[100],
    borderColor:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.brand.sunrise[700]
        : PEAK_DARK_THEME.extra.yellow[100],
  },
  critical: {
    color:
      theme.palette.mode === 'light'
        ? PEAK_DARK_THEME.error.dark
        : theme.palette.error.contrastText,
    borderColor:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.extra.red[50]
        : PEAK_DARK_THEME.error.dark,
    backgroundColor:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.extra.red[50]
        : PEAK_DARK_THEME.error.dark,
    transition: 'none',
  },
  down: {
    color:
      theme.palette.mode === 'light'
        ? PEAK_DARK_THEME.error.dark
        : theme.palette.error.contrastText,
    borderColor:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.extra.red[50]
        : PEAK_DARK_THEME.error.dark,
    backgroundColor:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.extra.red[50]
        : PEAK_DARK_THEME.error.dark,
    transition: 'none',
  },
});
