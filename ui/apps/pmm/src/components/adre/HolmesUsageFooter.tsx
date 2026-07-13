import { Box, Tooltip, Typography } from '@mui/material';
import { FC } from 'react';
import {
  holmesUsageSummaryLine,
  holmesUsageTooltip,
  type HolmesUsageDisplay,
} from 'utils/holmesUsageFormat';

export const HolmesUsageFooter: FC<{ usage: HolmesUsageDisplay; align?: 'left' | 'right' }> = ({
  usage,
  align = 'right',
}) => {
  const line = holmesUsageSummaryLine(usage);
  if (!line) return null;
  const tip = holmesUsageTooltip(usage);
  return (
    <Box sx={{ mt: 0.5, textAlign: align }}>
      <Tooltip title={tip || line} placement="top">
        <Typography variant="caption" color="text.secondary" component="span" sx={{ cursor: 'default' }}>
          {line}
        </Typography>
      </Tooltip>
    </Box>
  );
};
