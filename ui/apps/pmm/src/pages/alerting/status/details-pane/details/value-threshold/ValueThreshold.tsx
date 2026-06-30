import { FC } from 'react';
import { Box, Skeleton, Stack, Typography } from '@mui/material';
import { Messages } from '../AlertDetailsTab.messages';
import { AlertRow } from '../../../AlertsPage.types';
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

// Beyond ~10x over/under, the percentage stops conveying anything useful — e.g. a restart
// detector (`mysql_global_status_uptime < bool 5`) reads as "94620% over". Past this point we
// drop the percent and show only the direction word, so the value/threshold stays readable.
const PERCENT_OFF_SCALE = 1000;

const ValueThreshold: FC<Props> = ({ alert }) => {
  const { data, isLoading } = useAlertValueThreshold(alert);

  if (isLoading) {
    return <Skeleton variant="rounded" width="100%" height={32} />;
  }

  // Non-threshold rules (boolean up/down checks) or unavailable eval -> hide.
  if (!data) {
    return <UnavailableText />;
  }

  const { value, threshold, direction, breaching, percent } = data;
  const isOver = value > threshold;

  // The bar spans whichever is larger so the threshold marker always fits. The value segment fills
  // up to the threshold; any overflow above the threshold is shown as a second segment. Colour is
  // driven by `breaching` (operator-aware) so a `<` rule reads red when the value is below threshold.
  const span = Math.max(value, threshold);
  const valueFraction =
    span > 0 ? (isOver ? threshold / span : value / span) : 0;
  const overflowFraction = isOver && span > 0 ? (value - threshold) / span : 0;
  const fillColor = breaching ? 'error.main' : 'primary.main';

  const suffix =
    direction === 'over'
      ? Messages.details.percentOver
      : Messages.details.percentUnder;
  const hasMeaningfulPercent = percent !== null && percent <= PERCENT_OFF_SCALE;
  const ratioLabel = hasMeaningfulPercent ? `${percent}${suffix}` : direction;

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
            width: toPercent(valueFraction),
            backgroundColor: fillColor,
          }}
        />
        {overflowFraction > 0 && (
          <Box
            sx={{
              width: toPercent(overflowFraction),
              backgroundColor: fillColor,
            }}
          />
        )}
      </Box>
    </Stack>
  );
};

export default ValueThreshold;
