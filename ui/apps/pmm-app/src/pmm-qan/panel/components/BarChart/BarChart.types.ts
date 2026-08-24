import type { ChartProps } from 'react-chartjs-2';
import type { ChartData, ChartOptions } from 'chart.js';
import type { GrafanaTheme2 } from '@grafana/data';

export type BarChartOrientation = 'vertical' | 'horizontal';
export interface BarChartProps extends Omit<ChartProps<'bar'>, 'type'> {
  orientation?: BarChartOrientation;
  barWidth?: number;
  showLegend?: boolean;
  data: ChartData<'bar', number[], unknown>;
}

export type GetDefaultOptionsProps = {
  options?: ChartOptions<'bar'>;
  orientation?: BarChartOrientation;
  barWidth?: number;
  showLegend?: boolean;
  theme: GrafanaTheme2;
};
