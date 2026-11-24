import type { BoxProps } from '@mui/material/Box';
import type { ChartColor } from '../../../design/chart-colors';

export interface SegmentMeterProps extends Omit<
  BoxProps,
  'children' | 'color'
> {
  /** Share of the total, from 0 to 1; any share above 0 fills at least one dash, and only 1 fills them all. */
  value: number;
  /**
   * Number of dashes the meter is split into.
   * @default 5
   */
  segments?: number;
  /**
   * Theme color of the filled dashes: `primary` or a `palette.charts` key.
   * @default 'primary'
   */
  color?: ChartColor;
  /**
   * What the meter measures, as screen readers announce it.
   * @default 'Share of total'
   */
  ariaLabel?: string;
  /** How screen readers read the value; defaults to the percentage, such as "12% of total". */
  valueText?: string;
  /** Hides the meter from screen readers, for when the percentage is already shown next to it. */
  decorative?: boolean;
}
