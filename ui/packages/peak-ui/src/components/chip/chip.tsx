import { forwardRef } from 'react';
import MuiChip from '@mui/material/Chip';
import HighlightOff from '@mui/icons-material/HighlightOff';
import type { ChipProps } from './chip.types';

const Chip = forwardRef<HTMLDivElement, ChipProps>(
  ({ deleteIcon, ...rest }, ref) => (
    <MuiChip
      ref={ref}
      {...rest}
      icon={undefined}
      deleteIcon={<span>{deleteIcon ?? <HighlightOff />}</span>}
    />
  )
);

Chip.displayName = 'Chip';

export default Chip;
