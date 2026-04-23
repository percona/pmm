import { forwardRef } from 'react';
import { Chip as MuiChip } from '@mui/material';
import { HighlightOff } from '@mui/icons-material';
import type { ChipProps } from './chip.types';

const Chip = forwardRef<HTMLDivElement, ChipProps>(
  ({ dataTestId, deleteIcon, ...rest }, ref) => (
    <MuiChip
      ref={ref}
      data-testid={dataTestId ?? 'chip'}
      {...rest}
      icon={undefined}
      deleteIcon={<span>{deleteIcon ?? <HighlightOff />}</span>}
    />
  )
);

Chip.displayName = 'Chip';

export default Chip;
