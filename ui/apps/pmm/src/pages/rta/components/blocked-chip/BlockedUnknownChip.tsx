import Stack from '@mui/material/Stack';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import { Chip, Tooltip } from '@percona/peak-ui';
import { FC } from 'react';
import { Messages } from './BlockedChip.messages';

export interface Props {
  // unattributed: the connection was waiting for a lock, but for a later statement than the one
  // shown. unreadable: the server says this statement is waiting for a lock, but PMM cannot read
  // that kind of lock on this instance, so the holder is unknown.
  reason: 'unattributed' | 'unreadable';
}

// Outlined rather than warning-coloured: neither case names a holder, and neither is counted as
// blocked.
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
