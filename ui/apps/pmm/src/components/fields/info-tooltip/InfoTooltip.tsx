import InfoOutlined from '@mui/icons-material/InfoOutlined';
import Box from '@mui/material/Box';
import { Tooltip } from '@percona/peak-ui';
import type { FC } from 'react';
import type { InfoTooltipProps } from './InfoTooltip.types';

export const InfoTooltip: FC<InfoTooltipProps> = ({ title }) => (
  <Tooltip title={title} describeChild>
    {/* Focusable so the tooltip also opens from the keyboard; it describes the field rather than naming it. */}
    <Box component="span" tabIndex={0} sx={{ display: 'inline-flex', ml: 0.5 }}>
      <InfoOutlined fontSize="inherit" />
    </Box>
  </Tooltip>
);
