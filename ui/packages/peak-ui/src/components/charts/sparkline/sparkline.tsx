import { forwardRef, useCallback, useMemo } from 'react';
import Box from '@mui/material/Box';
import { alpha, useTheme } from '@mui/material/styles';
import { lineClasses } from '@mui/x-charts/LineChart';
import { SparkLineChart } from '@mui/x-charts/SparkLineChart';
import { getChartColor } from '../../../design/chart-colors';
import { SPARKLINE_DEFAULT_HEIGHT } from '../charts.constants';
import {
  SPARKLINE_HIDDEN,
  SPARKLINE_MARGIN,
  SPARKLINE_SLOTS,
} from './sparkline.constants';
import type { SparklineProps } from './sparkline.types';
import {
  defaultTimeFormatter,
  defaultValueFormatter,
  summarize,
} from './sparkline.utils';

const Sparkline = forwardRef<HTMLDivElement, SparklineProps>(
  (
    {
      data,
      variant = 'bar',
      height = SPARKLINE_DEFAULT_HEIGHT,
      width,
      color = 'primary',
      valueFormatter = defaultValueFormatter,
      timeFormatter = defaultTimeFormatter,
      ariaLabel,
      decorative = false,
      sx,
      ...rest
    },
    ref
  ) => {
    const theme = useTheme();
    const markColor = getChartColor(theme, color);
    const a11y = decorative
      ? SPARKLINE_HIDDEN
      : { 'aria-label': ariaLabel ?? summarize(data, valueFormatter) };

    // A single point has no time span for a line, so it is drawn as one bar.
    const bars = variant === 'bar' || data.length === 1;
    const values = useMemo(() => data.map((point) => point.value), [data]);
    const xAxis = useMemo(
      () => ({
        scaleType: bars ? ('band' as const) : ('time' as const),
        data: data.map((point) => new Date(point.timestamp)),
        valueFormatter: timeFormatter,
      }),
      [data, bars, timeFormatter]
    );
    const chartValueFormatter = useCallback(
      (value: number | null) => (value === null ? '' : valueFormatter(value)),
      [valueFormatter]
    );

    if (data.length === 0) {
      return (
        <Box
          ref={ref}
          height={height}
          width={width}
          {...(decorative ? a11y : { role: 'img', ...a11y })}
          {...rest}
          sx={sx}
        />
      );
    }

    if (values.every((value) => value === 0)) {
      return (
        <Box
          ref={ref}
          height={height}
          width={width}
          display="flex"
          alignItems="flex-end"
          data-flat
          {...(decorative ? a11y : { role: 'img', ...a11y })}
          {...rest}
          sx={sx}
        >
          <Box width="100%" borderBottom={`2px dotted ${markColor}`} />
        </Box>
      );
    }

    return (
      <Box
        ref={ref}
        width={width}
        minWidth={0}
        {...(decorative ? a11y : { role: 'group', ...a11y })}
        {...rest}
        sx={[
          {
            borderRadius: `${theme.shape.borderRadiusXs}px`,
            '&:has(:focus-visible)': {
              outline: `2px solid ${theme.palette.primary.main}`,
              outlineOffset: 2,
            },
          },
          ...(Array.isArray(sx) ? sx : [sx]),
        ]}
      >
        <SparkLineChart
          data={values}
          height={height}
          width={width}
          color={markColor}
          margin={SPARKLINE_MARGIN}
          showTooltip
          slots={SPARKLINE_SLOTS}
          showHighlight
          skipAnimation
          disableKeyboardNavigation={decorative}
          valueFormatter={chartValueFormatter}
          xAxis={xAxis}
          plotType={bars ? 'bar' : 'line'}
          area={!bars}
          curve="linear"
          sx={
            bars
              ? undefined
              : { [`& .${lineClasses.area}`]: { fill: alpha(markColor, 0.2) } }
          }
        />
      </Box>
    );
  }
);

Sparkline.displayName = 'Sparkline';

export default Sparkline;
