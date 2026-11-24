import type { BoxProps } from '@mui/material/Box';
import type { ChartColor } from '../../../design/chart-colors';

export type SparklineVariant = 'bar' | 'area';

export interface SparklinePoint {
  /** When the value was measured: a Date, epoch milliseconds or an ISO string. */
  timestamp: Date | number | string;
  /** The measured value at that time. */
  value: number;
}

export interface SparklineProps extends Omit<
  BoxProps,
  'children' | 'color' | 'height' | 'width' | 'ref'
> {
  /** The points to draw, oldest first. */
  data: SparklinePoint[];
  /**
   * bar: one bar per point, for counts per interval; area: a filled line, for a level over time. A single point is always a bar.
   * @default 'bar'
   */
  variant?: SparklineVariant;
  /**
   * Height of the chart in pixels.
   * @default 48
   */
  height?: number;
  /** Fixed width in pixels; without it the chart fills its container. */
  width?: number;
  /**
   * Theme color of the marks: `primary` or a `palette.charts` key.
   * @default 'primary'
   */
  color?: ChartColor;
  /** Turns a value into the text shown in the tooltip and the screen reader summary. */
  valueFormatter?: (value: number) => string;
  /** Turns a point's time into the text shown in the tooltip. */
  timeFormatter?: (time: Date) => string;
  /** What screen readers announce for the chart; without it, a summary of the latest, lowest and highest value. */
  ariaLabel?: string;
  /** Hides the chart from screen readers and the keyboard, for when the same numbers are already shown next to it. */
  decorative?: boolean;
}
