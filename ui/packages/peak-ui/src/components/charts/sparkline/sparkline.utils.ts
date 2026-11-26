import { CHARTS_EMPTY_TEXT } from '../charts.constants';
import { SPARKLINE_NUMBER_FORMAT } from './sparkline.constants';
import type { SparklinePoint } from './sparkline.types';

export const defaultValueFormatter = (value: number) =>
  SPARKLINE_NUMBER_FORMAT.format(value);
export const defaultTimeFormatter = (time: Date) => time.toLocaleString();

export const summarize = (
  data: SparklinePoint[],
  valueFormatter: (value: number) => string
) => {
  if (data.length === 0) return CHARTS_EMPTY_TEXT;
  const values = data.map((point) => point.value);
  return `Latest ${valueFormatter(values[values.length - 1])}, lowest ${valueFormatter(
    Math.min(...values)
  )}, highest ${valueFormatter(Math.max(...values))}`;
};
