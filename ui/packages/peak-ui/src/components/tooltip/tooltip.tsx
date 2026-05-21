import { forwardRef } from 'react';
import MuiTooltip from '@mui/material/Tooltip';
import type { TooltipProps } from './tooltip.types';

const Tooltip = forwardRef<HTMLDivElement, TooltipProps>(({ children, ...rest }, ref) => (
  <MuiTooltip ref={ref} {...rest}>
    {children}
  </MuiTooltip>
));

Tooltip.displayName = 'Tooltip';

export default Tooltip;
