import { Theme } from '@mui/material/styles';
import { PEAK_DARK_THEME } from '@pmm/shared';
import { CSSProperties } from 'react';
import { RealTimeQueryState } from 'types/real-time.types';

export const getStyles = (
  theme: Theme,
  state: RealTimeQueryState
): CSSProperties => {
  if (state === 'Blocked') {
    return theme.palette.mode === 'dark'
      ? {
          color: PEAK_DARK_THEME.extra.red[200],
          borderColor: PEAK_DARK_THEME.extra.red[200],
          backgroundColor: 'transparent',
        }
      : {
          color: PEAK_DARK_THEME.extra.red[500],
          borderColor: PEAK_DARK_THEME.extra.red[500],
          backgroundColor: 'transparent',
        };
  }

  if (state === 'Running') {
    return theme.palette.mode === 'dark'
      ? {
          color: PEAK_DARK_THEME.brand.sky[400],
          borderColor: PEAK_DARK_THEME.brand.sky[400],
        }
      : {
          color: PEAK_DARK_THEME.brand.sky[600],
          borderColor: PEAK_DARK_THEME.brand.sky[600],
        };
  }

  if (state === 'Sorting result' || state === 'Waiting') {
    return theme.palette.mode === 'dark'
      ? {
          color: PEAK_DARK_THEME.extra.yellow[100],
          borderColor: PEAK_DARK_THEME.extra.yellow[100],
        }
      : {
          color: PEAK_DARK_THEME.brand.sunrise[700],
          borderColor: PEAK_DARK_THEME.brand.sunrise[700],
        };
  }

  return {};
};
