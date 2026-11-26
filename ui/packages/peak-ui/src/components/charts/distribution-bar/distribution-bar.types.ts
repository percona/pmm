import type { BoxProps } from '@mui/material/Box';
import type { ChartColor } from '../../../design/chart-colors';

export interface DistributionBarSegment {
  /** Stable key of the part, unique within the bar. */
  id: string;
  /** Name of the part, shown in the legend. */
  label: string;
  /** Amount in any unit; each part's share is its value over the sum of all values. */
  value: number;
  /** Theme color of the part; defaults to the `palette.charts` color at its position. */
  color?: ChartColor;
}

export interface DistributionBarProps extends Omit<BoxProps, 'children'> {
  /** The parts of the whole, in the order they are drawn from the left. */
  segments: DistributionBarSegment[];
  /** Turns a part's share (0 to 1) into the text shown in the bar and the legend; defaults to a percentage. */
  valueFormatter?: (share: number, segment: DistributionBarSegment) => string;
  /** Hides the legend visually; screen readers still read every part and its share. */
  hideLegend?: boolean;
  /** What the distribution shows, such as "Query time"; names the legend for screen readers. */
  ariaLabel?: string;
}
