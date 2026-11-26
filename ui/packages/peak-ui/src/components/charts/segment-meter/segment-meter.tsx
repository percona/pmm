import { forwardRef } from 'react';
import Box from '@mui/material/Box';
import { useTheme } from '@mui/material/styles';
import { getChartColor } from '../../../design/chart-colors';
import { formatShare, shareToPercent } from '../charts.utils';
import type { SegmentMeterProps } from './segment-meter.types';

const getFilledSegments = (share: number, segments: number): number => {
  if (share <= 0) return 0;
  if (share >= 1) return segments;
  return Math.max(1, Math.min(segments - 1, Math.round(share * segments)));
};

const SegmentMeter = forwardRef<HTMLDivElement, SegmentMeterProps>(
  (
    {
      value,
      segments = 5,
      color = 'primary',
      ariaLabel = 'Share of total',
      valueText,
      decorative = false,
      ...rest
    },
    ref
  ) => {
    const theme = useTheme();
    const share = Math.min(1, Math.max(0, value));
    const filled = getFilledSegments(share, segments);
    const fill = getChartColor(theme, color);
    const a11y = decorative
      ? { 'aria-hidden': true }
      : {
          role: 'meter',
          'aria-label': ariaLabel,
          'aria-valuemin': 0,
          'aria-valuemax': 100,
          'aria-valuenow': shareToPercent(share),
          'aria-valuetext': valueText ?? `${formatShare(share)} of total`,
        };

    return (
      <Box
        ref={ref}
        display="inline-flex"
        alignItems="center"
        gap={0.5}
        {...a11y}
        {...rest}
      >
        {Array.from({ length: segments }, (_, index) => (
          <Box
            key={index}
            data-filled={index < filled}
            sx={{
              width: 12,
              height: 4,
              borderRadius: `${theme.shape.borderRadiusXs}px`,
              bgcolor: index < filled ? fill : theme.palette.dividers.divider,
            }}
          />
        ))}
      </Box>
    );
  }
);

SegmentMeter.displayName = 'SegmentMeter';

export default SegmentMeter;
