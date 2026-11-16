import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Tooltip } from '@percona/peak-ui';
import type { FC } from 'react';
import { SERVICE_STATUSES } from '../../Inventory.constants';
import { Messages } from '../../Inventory.messages';
import { StatusChip } from '../status-chip';

const StatusInfo: FC = () => (
  <Stack
    gap={1}
    py={0.5}
    // Default chips use the page palette, which the inverted tooltip hides
    sx={{
      '& .MuiChip-colorDefault': {
        color: 'inherit',
        bgcolor: 'transparent',
        border: 1,
        borderColor: 'currentColor',
      },
    }}
  >
    {SERVICE_STATUSES.map((status) => (
      <Stack key={status} direction="row" alignItems="center" gap={1}>
        <StatusChip status={status} />
        <Typography variant="body2">{Messages.statusInfo[status]}</Typography>
      </Stack>
    ))}
  </Stack>
);

export const StatusHeader: FC = () => (
  <Stack direction="row" alignItems="center" gap={0.5}>
    {Messages.columns.status}
    <Tooltip title={<StatusInfo />}>
      <InfoOutlinedIcon
        fontSize="small"
        color="action"
        tabIndex={0}
        // The header label sorts the column on click
        onClick={(event) => event.stopPropagation()}
        data-testid="status-info-icon"
      />
    </Tooltip>
  </Stack>
);
