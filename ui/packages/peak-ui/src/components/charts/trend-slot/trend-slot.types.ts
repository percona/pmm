import type { ReactNode } from 'react';
import type { SparklineProps, SparklineVariant } from '../sparkline';

export interface TrendSlotProps {
  /** The trend to draw; without it, the slot shows nothing once loaded. */
  sparkline?: SparklineProps;
  /** Shows a placeholder at the trend's height while it loads. */
  loading: boolean;
  /** What shows in place of the trend when the sparkline has no points. */
  emptyText: ReactNode;
  /** How the trend is drawn unless the sparkline sets its own variant. */
  variant: SparklineVariant;
  /** The trend's height, and its placeholder's, unless the sparkline sets its own. */
  defaultHeight?: number;
}
