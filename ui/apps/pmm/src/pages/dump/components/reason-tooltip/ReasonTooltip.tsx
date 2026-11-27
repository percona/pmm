import Box from '@mui/material/Box';
import Tooltip from '@mui/material/Tooltip';
import type { FC, PropsWithChildren } from 'react';

interface Props extends PropsWithChildren {
  reason?: string;
}

// The span lets a disabled control, which fires no mouse events, show its tooltip
export const ReasonTooltip: FC<Props> = ({ reason, children }) => (
  <Tooltip title={reason ?? ''}>
    <Box component="span" sx={{ display: 'inline-flex' }}>
      {children}
    </Box>
  </Tooltip>
);
