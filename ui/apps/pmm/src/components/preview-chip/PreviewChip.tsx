import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import type { FC } from 'react';
import {
  PREVIEW_CHIP_LABEL,
  PREVIEW_LEVEL_TOOLTIP,
} from './PreviewChip.constants';
import { visuallyHidden } from './PreviewChip.styles';
import type { PreviewChipProps } from './PreviewChip.types';

const PreviewChip: FC<PreviewChipProps> = ({ level }) => (
  <Stack flex={8} alignItems="flex-start">
    <Tooltip title={PREVIEW_LEVEL_TOOLTIP[level]} describeChild arrow>
      <Chip
        data-testid="preview-chip"
        label={
          <>
            <span aria-hidden="true">{PREVIEW_CHIP_LABEL}</span>
            <Box component="span" sx={visuallyHidden}>
              {PREVIEW_LEVEL_TOOLTIP[level]}
            </Box>
          </>
        }
        size="small"
        color="warning"
        variant="outlined"
      />
    </Tooltip>
  </Stack>
);

export default PreviewChip;
