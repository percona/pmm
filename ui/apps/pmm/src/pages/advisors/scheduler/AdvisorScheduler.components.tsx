import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { ADVISOR_TECHNOLOGY } from 'lib/constants';
import { type FC, type ReactNode } from 'react';
import { AdvisorTechnology } from 'types/advisors.types';

export const MONO_FONT = 'Roboto Mono, monospace';

export const TECHNOLOGY_COLOR: Record<AdvisorTechnology, string> = {
  [AdvisorTechnology.mysql]: '#E65A15',
  [AdvisorTechnology.postgresql]: '#2486FF',
  [AdvisorTechnology.mongodb]: '#1FA23A',
  [AdvisorTechnology.unspecified]: 'transparent',
};

// a schedule's color; dashed and hollow while it is paused
export const Swatch: FC<{ color: string; paused?: boolean }> = ({
  color,
  paused,
}) => (
  <Box
    component="span"
    sx={{
      flex: 'none',
      width: 10,
      height: 10,
      borderRadius: '3px',
      border: `1.5px ${paused ? 'dashed' : 'solid'} ${color}`,
      bgcolor: paused ? 'transparent' : color,
    }}
  />
);

export const StatusDot: FC<{ color: string }> = ({ color }) => (
  <Box
    component="span"
    sx={{
      flex: 'none',
      width: 8,
      height: 8,
      borderRadius: '50%',
      bgcolor: color,
    }}
  />
);

export const TechnologyLabel: FC<{ technology: AdvisorTechnology }> = ({
  technology,
}) => (
  <Stack direction="row" alignItems="center" gap={0.75} component="span">
    <StatusDot color={TECHNOLOGY_COLOR[technology]} />
    {ADVISOR_TECHNOLOGY[technology]}
  </Stack>
);

interface TwoLineProps {
  primary: ReactNode;
  secondary?: ReactNode;
  muted?: boolean;
  testId?: string;
}

// a table cell's value with a smaller line under it
export const TwoLine: FC<TwoLineProps> = ({
  primary,
  secondary,
  muted,
  testId,
}) => (
  <Stack data-testid={testId}>
    <Typography
      variant="body2"
      component="span"
      color={muted ? 'text.disabled' : undefined}
      sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}
    >
      {primary}
    </Typography>
    {secondary && (
      <Typography variant="caption" color="text.secondary">
        {secondary}
      </Typography>
    )}
  </Stack>
);
