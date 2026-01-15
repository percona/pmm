import { Theme } from '@mui/material';
import { PEAK_DARK_THEME, PEAK_LIGHT_THEME } from '@pmm/shared';

export const getStyles = ({ palette: { mode, background } }: Theme) => ({
  icon: {
    width: 20,
    height: 20,

    '.background': {
      fill: background.default,
    },
    '.status-at-risk': {
      fill:
        mode === 'light'
          ? PEAK_LIGHT_THEME.brand.sunrise[700]
          : PEAK_DARK_THEME.extra.yellow[100],
    },
    '.status-down': {
      fill:
        mode === 'light'
          ? PEAK_LIGHT_THEME.extra.red[500]
          : PEAK_DARK_THEME.extra.red[200],
    },
    '.status-updating': {
      fill:
        mode === 'light'
          ? PEAK_LIGHT_THEME.brand.sky[600]
          : PEAK_DARK_THEME.brand.sky[400],
    },
  },
});
