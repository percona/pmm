import { getContrastRatio, type Theme } from '@mui/material/styles';
import type { ChartPalette } from './themes/base/BaseTheme';

/** A theme color for chart marks: `primary`, or one of the `palette.charts` series colors. */
export type ChartColor = 'primary' | keyof ChartPalette;

export const CHART_SERIES_COLORS = [
  'chart1',
  'chart2',
  'chart3',
  'chart4',
  'chart5',
  'chart6',
  'chart7',
] as const satisfies readonly ChartColor[];

export const getChartColor = (theme: Theme, color: ChartColor): string =>
  color === 'primary'
    ? theme.palette.primary.main
    : theme.palette.charts[color];

export const getReadableTextColor = (
  theme: Theme,
  background: string
): string => {
  const { black, white } = theme.palette.common;
  return getContrastRatio(white, background) >=
    getContrastRatio(black, background)
    ? white
    : black;
};
