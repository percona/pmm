import { forwardRef, useId } from 'react';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Skeleton from '@mui/material/Skeleton';
import Typography from '@mui/material/Typography';
import { CHARTS_EMPTY_TEXT } from '../charts.constants';
import TrendSlot from '../trend-slot';
import { STAT_TILE_SIZES } from './stat-tile.constants';
import type { StatTileProps } from './stat-tile.types';

const StatTile = forwardRef<HTMLDivElement, StatTileProps>(
  (
    {
      label,
      value,
      sparkline,
      loading = false,
      emptyText = CHARTS_EMPTY_TEXT,
      size = 'medium',
      sx,
      ...rest
    },
    ref
  ) => {
    const labelId = useId();
    const { p, labelMinWidth, valueVariant, sparklineHeight, trendWidth } =
      STAT_TILE_SIZES[size];

    return (
      <Paper
        ref={ref}
        variant="outlined"
        role="group"
        aria-labelledby={labelId}
        aria-busy={loading}
        sx={[
          { p, display: 'flex', gap: p, alignItems: 'center', minWidth: 0 },
          ...(Array.isArray(sx) ? sx : [sx]),
        ]}
        {...rest}
      >
        <Box
          display="flex"
          flexDirection="column"
          flex={trendWidth && 1}
          minWidth={labelMinWidth}
        >
          <Typography id={labelId} variant="caption">
            {label}
          </Typography>
          <Typography variant={valueVariant} component="p" noWrap>
            {loading ? <Skeleton width={80} /> : value}
          </Typography>
        </Box>
        {(loading || sparkline) && (
          <Box flex={trendWidth ? `0 1 ${trendWidth}px` : 1} minWidth={0}>
            <TrendSlot
              sparkline={sparkline}
              loading={loading}
              emptyText={emptyText}
              variant="area"
              defaultHeight={sparklineHeight}
            />
          </Box>
        )}
      </Paper>
    );
  }
);

StatTile.displayName = 'StatTile';

export default StatTile;
