import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { Chip, Tooltip, type ChipProps } from '@percona/peak-ui';
import { formatTimestamp } from '@pmm-extensions/framework';
import type { FC } from 'react';
import { Messages } from '../hub.messages';
import type { HubHealth } from '../hub.types';
import { focusableText } from './hub.sx';
import { formatSince } from './hub.time';

// Filled for the states that need someone, outlined for healthy, grey for no data.
const CHIP: Record<HubHealth, Pick<ChipProps, 'color' | 'variant'>> = {
  healthy: { color: 'success', variant: 'outlined' },
  degraded: { color: 'warning', variant: 'filled' },
  down: { color: 'error', variant: 'filled' },
  noData: { color: 'default', variant: 'filled' },
};

export interface HealthChipProps {
  health: HubHealth;
  /** ISO time the node has been in this state since; shown as "for 12m". */
  since?: string;
  /** Tooltip on the duration; the exact time when absent. */
  detail?: string;
}

/** A health word on a Peak Chip, with how long a problem has lasted beside it. */
export const HealthChip: FC<HealthChipProps> = ({ health, since, detail }) => {
  const word = Messages.health[health];
  const age = health === 'healthy' ? null : formatSince(since);

  return (
    <Box
      component="span"
      sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75 }}
    >
      <Chip
        size="small"
        label={word}
        data-testid="hub-health"
        {...CHIP[health]}
      />
      {age != null && (
        <Tooltip
          title={
            detail ??
            Messages.healthSince(word, formatTimestamp(since)?.title ?? '')
          }
          describeChild
        >
          <Typography
            variant="caption"
            color="text.secondary"
            noWrap
            tabIndex={0}
            sx={focusableText}
          >
            {Messages.healthFor(age)}
          </Typography>
        </Tooltip>
      )}
    </Box>
  );
};
