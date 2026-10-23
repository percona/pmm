import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { Chip, Tooltip, type ChipProps } from '@percona/peak-ui';
import { formatTimestamp } from '@pmm-extensions/framework';
import type { FC } from 'react';
import { Messages } from '../hub.messages';
import type { HubCondition, HubHealth } from '../hub.types';
import { focusableText } from './hub.sx';
import { formatSince } from './hub.time';

// Filled for the states that need someone, outlined for healthy and planned work, grey for no data.
const CHIP: Record<HubHealth, Pick<ChipProps, 'color' | 'variant'>> = {
  healthy: { color: 'success', variant: 'outlined' },
  degraded: { color: 'warning', variant: 'filled' },
  down: { color: 'error', variant: 'filled' },
  upgrading: { color: 'info', variant: 'outlined' },
  noData: { color: 'default', variant: 'filled' },
};

export interface HealthChipProps {
  health: HubHealth;
  /** ISO time the node has been in this state since; shown as "for 12m". */
  since?: string;
  /** What makes it other than healthy, the deciding one first; the first shows beneath the chip. */
  conditions?: HubCondition[];
}

/** Each condition with the rule it breaks, then since when, for the reason line's tooltip. */
const ConditionsDetail: FC<{ conditions: HubCondition[]; since?: string }> = ({
  conditions,
  since,
}) => (
  <>
    {conditions.map(({ text, rule }, index) => (
      <Box
        key={`${index}:${text}`}
        component="span"
        sx={{ display: 'block', mb: 0.5 }}
      >
        <Box
          component="span"
          sx={{ display: 'block', fontWeight: 'fontWeightMedium' }}
        >
          {text}
        </Box>
        <Box component="span" sx={{ display: 'block' }}>
          {rule}
        </Box>
      </Box>
    ))}
    {since}
  </>
);

/** A health word on a Peak Chip, how long a problem has lasted beside it, and why beneath it. */
export const HealthChip: FC<HealthChipProps> = ({
  health,
  since,
  conditions = [],
}) => {
  const word = Messages.health[health];
  const age = health === 'healthy' ? null : formatSince(since);
  const exact = formatTimestamp(since)?.title;
  const sinceText = exact ? Messages.healthSince(word, exact) : undefined;
  const [first] = conditions;

  return (
    <Box
      component="span"
      // Bounded by its cell, so a long age ellipsizes and a long reason wraps instead of running past the edge.
      sx={{
        display: 'inline-flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        gap: 0.25,
        minWidth: 0,
        maxWidth: '100%',
      }}
    >
      <Box
        component="span"
        sx={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 0.75,
          minWidth: 0,
          maxWidth: '100%',
        }}
      >
        <Chip
          size="small"
          label={word}
          data-testid="hub-health"
          sx={{ flexShrink: 0 }}
          {...CHIP[health]}
        />
        {age != null && (
          <Tooltip title={sinceText ?? ''} describeChild>
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
      {first && (
        <Tooltip
          title={<ConditionsDetail conditions={conditions} since={sinceText} />}
          describeChild
        >
          <Typography
            variant="caption"
            color="text.secondary"
            tabIndex={0}
            data-testid="hub-health-reason"
            sx={[focusableText, { whiteSpace: 'normal' }]}
          >
            {first.text}
            {conditions.length > 1 &&
              Messages.conditionsMore(conditions.length - 1)}
          </Typography>
        </Tooltip>
      )}
    </Box>
  );
};
