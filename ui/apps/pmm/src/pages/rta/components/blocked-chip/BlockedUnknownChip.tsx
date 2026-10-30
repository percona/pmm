import Stack from '@mui/material/Stack';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import { Chip, Tooltip } from '@percona/peak-ui';
import { FC } from 'react';
import { Messages } from './BlockedChip.messages';

export interface Props {
  // unattributed: the wait was in a later statement; unreadable: PMM cannot read this lock type.
  reason: 'unattributed' | 'unreadable';
}

// Outlined, not warning-coloured: neither case names a holder or counts as blocked.
const BlockedUnknownChip: FC<Props> = ({ reason }) => (
  <Tooltip
    title={
      reason === 'unreadable'
        ? Messages.tooltipUnreadable
        : Messages.tooltipUnattributed
    }
    arrow
  >
    <Chip
      variant="outlined"
      data-testid="blocked-unknown-chip"
      data-reason={reason}
      sx={{ flexShrink: 0 }}
      label={
        <Stack direction="row" alignItems="center" gap={0.5}>
          <HelpOutlineIcon fontSize="small" />
          {Messages.blockedUnattributed}
        </Stack>
      }
    />
  </Tooltip>
);

export default BlockedUnknownChip;
