import type { PaperProps } from '@mui/material/Paper';
import type { ReactNode } from 'react';
import type { SparklineProps } from '../sparkline';

export type StatTileSize = 'small' | 'medium';

export interface StatTileProps extends Omit<PaperProps, 'children'> {
  /** What the tile measures, such as "Operations". */
  label: ReactNode;
  /** The current value, already formatted with its unit. */
  value: ReactNode;
  /** A trend of the value over time, drawn next to it; area variant by default. */
  sparkline?: SparklineProps;
  /** Shows placeholders while the value loads. */
  loading?: boolean;
  /**
   * What shows in place of the trend when the sparkline has no points.
   * @default 'No data'
   */
  emptyText?: ReactNode;
  /**
   * small: a dense row of tiles, such as a cluster summary; medium: an overview page. Sets the padding, the value size and the trend's size.
   * @default 'medium'
   */
  size?: StatTileSize;
}
