import InfoOutlined from '@mui/icons-material/InfoOutlined';
import Stack from '@mui/material/Stack';
import { Tooltip } from '@percona/peak-ui';
import type { FC } from 'react';
import type { FieldLabelProps } from './FieldLabel.types';

export const FieldLabel: FC<FieldLabelProps> = ({ label, tooltip }) =>
  tooltip ? (
    <Stack component="span" direction="row" alignItems="center" gap={0.5}>
      {label}
      <Tooltip title={tooltip}>
        <InfoOutlined fontSize="inherit" />
      </Tooltip>
    </Stack>
  ) : (
    label
  );
