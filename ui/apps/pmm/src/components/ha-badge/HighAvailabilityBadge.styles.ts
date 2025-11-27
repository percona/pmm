import { Theme } from '@mui/material/styles';
import { PEAK_DARK_THEME, PEAK_LIGHT_THEME } from '@pmm/shared';

export const getStyles = (theme: Theme) => ({
  healthy: {
    color: theme.palette.text.primary,
    borderColor: theme.palette.text.primary,
  },
  'at-risk': {
    color:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.brand.sunrise[700]
        : PEAK_DARK_THEME.extra.yellow[100],
    borderColor:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.brand.sunrise[700]
        : PEAK_DARK_THEME.extra.yellow[100],
  },
  down: {
    color:
      theme.palette.mode === 'light'
        ? PEAK_DARK_THEME.error.dark
        : theme.palette.error.contrastText,
    backgroundColor:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.extra.red[50]
        : PEAK_DARK_THEME.error.dark,
    transition: 'none',
  },
  updating: {
    color:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.brand.sky[600]
        : PEAK_DARK_THEME.brand.sky[400],
    borderColor:
      theme.palette.mode === 'light'
        ? PEAK_LIGHT_THEME.brand.sky[600]
        : PEAK_DARK_THEME.brand.sky[400],
  },
});
