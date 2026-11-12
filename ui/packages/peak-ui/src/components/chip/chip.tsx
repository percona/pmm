import { forwardRef } from 'react';
import MuiChip from '@mui/material/Chip';
import HighlightOff from '@mui/icons-material/HighlightOff';
import type { ChipProps } from './chip.types';

const Chip = forwardRef<HTMLDivElement, ChipProps>(({ deleteIcon, onDelete, ...rest }, ref) => (
  <MuiChip
    ref={ref}
    {...rest}
    onDelete={onDelete}
    deleteIcon={onDelete ? <span>{deleteIcon ?? <HighlightOff />}</span> : undefined}
  />
));

Chip.displayName = 'Chip';

export default Chip;
