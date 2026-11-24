import { forwardRef } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import visuallyHidden from '@mui/utils/visuallyHidden';
import {
  CHART_SERIES_COLORS,
  getChartColor,
  getReadableTextColor,
} from '../../../design/chart-colors';
import { TABULAR_NUMS } from '../charts.constants';
import { formatShare } from '../charts.utils';
import { LABEL_MIN_SHARE } from './distribution-bar.constants';
import type { DistributionBarProps } from './distribution-bar.types';

const DistributionBar = forwardRef<HTMLDivElement, DistributionBarProps>(
  (
    {
      segments,
      valueFormatter = formatShare,
      hideLegend = false,
      ariaLabel,
      ...rest
    },
    ref
  ) => {
    const theme = useTheme();
    const values = segments.map((segment) => Math.max(0, segment.value));
    const total = values.reduce((sum, value) => sum + value, 0);
    const parts = segments.map((segment, index) => {
      const share = total > 0 ? values[index] / total : 0;
      const color = getChartColor(
        theme,
        segment.color ?? CHART_SERIES_COLORS[index % CHART_SERIES_COLORS.length]
      );
      return { segment, share, color, text: valueFormatter(share, segment) };
    });

    return (
      <Box ref={ref} display="flex" flexDirection="column" gap={1} {...rest}>
        <Box
          aria-hidden
          sx={{
            display: 'flex',
            gap: '2px',
            height: 32,
            borderRadius: `${theme.shape.borderRadiusSm}px`,
            overflow: 'hidden',
            bgcolor: total > 0 ? undefined : theme.palette.dividers.divider,
          }}
        >
          {parts
            .filter(({ share }) => share > 0)
            .map(({ segment, share, color, text }) => (
              <Box
                key={segment.id}
                data-segment={segment.id}
                title={`${segment.label}: ${text}`}
                sx={{
                  flexGrow: share,
                  flexBasis: 0,
                  minWidth: 2,
                  bgcolor: color,
                  color: getReadableTextColor(theme, color),
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  overflow: 'hidden',
                }}
              >
                {share >= LABEL_MIN_SHARE && (
                  <Typography variant="caption" noWrap sx={TABULAR_NUMS}>
                    {text}
                  </Typography>
                )}
              </Box>
            ))}
        </Box>
        <Box
          component="ul"
          aria-label={ariaLabel}
          sx={
            hideLegend
              ? visuallyHidden
              : {
                  listStyle: 'none',
                  m: 0,
                  p: 0,
                  display: 'flex',
                  flexWrap: 'wrap',
                  columnGap: 2,
                  rowGap: 0.5,
                }
          }
        >
          {parts.map(({ segment, color, text }) => (
            <Box
              component="li"
              key={segment.id}
              display="flex"
              alignItems="center"
              gap={0.75}
            >
              <Box
                aria-hidden
                sx={{
                  width: 10,
                  height: 10,
                  borderRadius: '50%',
                  bgcolor: color,
                  flexShrink: 0,
                }}
              />
              <Typography variant="caption">{segment.label}</Typography>{' '}
              <Typography variant="caption" fontWeight={600} sx={TABULAR_NUMS}>
                {text}
              </Typography>
            </Box>
          ))}
        </Box>
      </Box>
    );
  }
);

DistributionBar.displayName = 'DistributionBar';

export default DistributionBar;
