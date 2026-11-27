import Stack from '@mui/material/Stack';
import type { FC } from 'react';
import { InfoTooltip } from '../info-tooltip';
import type { FieldLabelProps } from './FieldLabel.types';

export const FieldLabel: FC<FieldLabelProps> = ({ label, tooltip }) =>
  tooltip ? (
    <Stack
      component="span"
      direction="row"
      alignItems="center"
      // Inline, so the required asterisk MUI appends stays on the same line.
      sx={{ display: 'inline-flex' }}
    >
      {label}
      <InfoTooltip title={tooltip} />
    </Stack>
  ) : (
    label
  );
