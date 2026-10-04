import Stack from '@mui/material/Stack';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import { Chip, Tooltip } from '@percona/peak-ui';
import { FC } from 'react';
import { Messages } from './BlockedChip.messages';

// The connection was waiting for a lock, but for a later statement than the one shown, so
// this refresh cannot say whether the statement shown was blocked. Outlined rather than
// warning-coloured: it is not evidence of waiting and is not counted as blocked.
const BlockedUnknownChip: FC = () => (
  <Tooltip title={Messages.tooltipUnattributed} arrow>
    <Chip
      variant="outlined"
      data-testid="blocked-unknown-chip"
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
