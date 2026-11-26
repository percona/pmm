import { forwardRef, useId } from 'react';
import Box from '@mui/material/Box';
import Skeleton from '@mui/material/Skeleton';
import Typography from '@mui/material/Typography';
import { CHARTS_EMPTY_TEXT, TABULAR_NUMS } from '../charts.constants';
import TrendSlot from '../trend-slot';
import type { MetricCardProps } from './metric-card.types';

const MetricCard = forwardRef<HTMLDivElement, MetricCardProps>(
  (
    {
      title,
      sparkline,
      values,
      loading = false,
      emptyText = CHARTS_EMPTY_TEXT,
      ...rest
    },
    ref
  ) => {
    const titleId = useId();

    return (
      <Box
        ref={ref}
        role="group"
        aria-labelledby={titleId}
        aria-busy={loading}
        display="flex"
        flexDirection="column"
        gap={1}
        pb={1.5}
        borderBottom={1}
        borderColor="divider"
        minWidth={0}
        {...rest}
      >
        <Typography id={titleId} variant="subtitle2" component="p">
          {title}
        </Typography>
        <TrendSlot
          sparkline={sparkline}
          loading={loading}
          emptyText={emptyText}
          variant="bar"
        />
        <Box
          component="dl"
          m={0}
          display="flex"
          justifyContent="space-between"
          gap={1}
        >
          {values.map(({ label, value, note }, index) => (
            <Box key={index} minWidth={0}>
              <Typography component="dt" variant="caption">
                {label}
              </Typography>
              <Typography
                component="dd"
                variant="subHead2"
                m={0}
                noWrap
                sx={TABULAR_NUMS}
              >
                {loading ? (
                  <Skeleton width={56} />
                ) : (
                  <>
                    {value}
                    {note != null && (
                      <Typography component="span" variant="caption" ml={0.75}>
                        {note}
                      </Typography>
                    )}
                  </>
                )}
              </Typography>
            </Box>
          ))}
        </Box>
      </Box>
    );
  }
);

MetricCard.displayName = 'MetricCard';

export default MetricCard;
