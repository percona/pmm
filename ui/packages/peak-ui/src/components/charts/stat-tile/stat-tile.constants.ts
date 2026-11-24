import type { TypographyVariant } from '@mui/material/styles';
import { SPARKLINE_DEFAULT_HEIGHT } from '../charts.constants';
import type { StatTileSize } from './stat-tile.types';

export const STAT_TILE_SIZES: Record<
  StatTileSize,
  {
    p: number;
    labelMinWidth?: number;
    valueVariant: TypographyVariant;
    sparklineHeight: number;
    trendWidth?: number;
  }
> = {
  small: { p: 1.5, valueVariant: 'h6', sparklineHeight: 34, trendWidth: 84 },
  medium: {
    p: 2,
    labelMinWidth: 112,
    valueVariant: 'h5',
    sparklineHeight: SPARKLINE_DEFAULT_HEIGHT,
  },
};
