import type { BoxProps } from '@mui/material/Box';
import type { ReactNode } from 'react';
import type { SparklineProps } from '../sparkline';

export interface MetricCardValue {
  /** What the number means, such as "Per second" or "Sum". */
  label: ReactNode;
  /** The number, already formatted with its unit. */
  value: ReactNode;
  /** Extra context after the number, such as its share of the total. */
  note?: ReactNode;
}

export interface MetricCardProps extends Omit<BoxProps, 'children' | 'title'> {
  /** Name of the metric, such as "Query time". */
  title: ReactNode;
  /** A trend of the metric over time, drawn under the title; bar variant by default. */
  sparkline?: SparklineProps;
  /** The summary numbers under the trend, in display order. */
  values: MetricCardValue[];
  /** Shows placeholders while the metric loads. */
  loading?: boolean;
  /**
   * What shows in place of the trend when the sparkline has no points.
   * @default 'No data'
   */
  emptyText?: ReactNode;
}
