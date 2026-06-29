import { FC } from 'react';
import { Box, Skeleton, Stack, Typography } from '@mui/material';
import { Messages } from './AlertDetails.messages';
import { AlertRow } from '../AlertsPage.types';
import { useAlertValueThreshold } from 'hooks/api/useAlertValueThreshold';
import UnavailableText from 'components/unavailable-text';

interface Props {
  alert: AlertRow;
}

const clampFraction = (fraction: number): number =>
  Math.min(Math.max(fraction, 0), 1);

const toPercent = (fraction: number): string =>
  `${clampFraction(fraction) * 100}%`;

// Round to at most 2 decimals and drop trailing zeros (2.6490066 -> "2.65", 80 -> "80").
const formatNumber = (value: number): string => `${Number(value.toFixed(2))}`;

const ValueThreshold: FC<Props> = ({ alert }) => {
  const { data, isLoading } = useAlertValueThreshold(alert);

  if (isLoading) {
    return <Skeleton variant="rounded" width="100%" height={32} />;
  }

  // Non-threshold rules (boolean up/down checks) or unavailable eval -> hide.
  if (!data) {
    return <UnavailableText />;
  }

  const { value, threshold, direction, percent } = data;
  const isOver = value > threshold;

  // The bar spans whichever is larger so the threshold marker always fits. When the
  // value is over the threshold, the blue segment fills up to the threshold and the red
  // segment represents the overflow.
  const span = Math.max(value, threshold);
  const blueFraction = span > 0 ? (isOver ? threshold / span : value / span) : 0;
  const redFraction = isOver && span > 0 ? (value - threshold) / span : 0;

  const suffix =
    direction === 'over'
      ? Messages.details.percentOver
      : Messages.details.percentUnder;
  const ratioLabel = percent !== null ? `${percent}${suffix}` : direction;

  return (
    <Stack spacing={1}>
      <Typography variant="body1">
        {formatNumber(value)} / {formatNumber(threshold)} ({ratioLabel})
      </Typography>
      <Box
        sx={{
          display: 'flex',
          height: 8,
          width: '100%',
          borderRadius: 4,
          overflow: 'hidden',
          border: 1,
          borderColor: 'divider',
          backgroundColor: 'action.hover',
        }}
      >
        <Box
          sx={{
            width: toPercent(blueFraction),
            backgroundColor: 'primary.main',
          }}
        />
        {redFraction > 0 && (
          <Box
            sx={{
              width: toPercent(redFraction),
              backgroundColor: 'error.main',
            }}
          />
        )}
      </Box>
    </Stack>
  );
};

export default ValueThreshold;
