import type { BoxProps } from '@mui/material/Box';
import type { ChartColor } from '../../../design/chart-colors';

export interface BarListItem {
  /** Name of the row, such as a latency range; unique within the list. */
  label: string;
  /** Size of the row, usually a share from 0 to 1; bars are drawn relative to the largest. */
  value: number;
}

export interface BarListProps extends Omit<BoxProps, 'children' | 'color'> {
  /** The rows, in display order. */
  items: BarListItem[];
  /** Turns a value into the text shown after its bar; defaults to a percentage. */
  valueFormatter?: (value: number) => string;
  /**
   * Theme color of the bars: `primary` or a `palette.charts` key.
   * @default 'primary'
   */
  color?: ChartColor;
  /**
   * What a row with no value shows in place of a number.
   * @default '—'
   */
  emptyLabel?: string;
  /**
   * What screen readers announce for a row with no value.
   * @default 'No data'
   */
  emptyText?: string;
}
