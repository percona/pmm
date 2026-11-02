import Stack from '@mui/material/Stack';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { Chip, Tooltip } from '@percona/peak-ui';
import { FC } from 'react';
import { BlockingTransaction } from 'types/rta.types';
import { soleBlockerOf } from 'pages/rta/overview/table/OverviewTable.utils';
import { Messages } from './BlockedChip.messages';

export interface Props {
  // One is named only when it is the whole answer; otherwise the chip shows the count.
  blockers: BlockingTransaction[];
}

const BlockedChip: FC<Props> = ({ blockers }) => {
  const sole = soleBlockerOf(blockers);
  // The lock graph and the statement list are read milliseconds apart.
  const holderUnknown = blockers.length === 0;

  const label = holderUnknown
    ? Messages.blockedUnknownHolder
    : sole
      ? Messages.blockedBy(sole.blockingConnId)
      : Messages.blockedByMany(blockers.length);

  const tooltip = holderUnknown
    ? Messages.tooltipUnknownHolder
    : sole
      ? Messages.tooltip(sole.blockingConnId)
      : Messages.tooltipMany(blockers.length);

  return (
    <Tooltip title={tooltip} arrow>
      <Chip
        color="warning"
        data-testid="blocked-chip"
        // Beside a width:100% query cell, shrinking would clip the connection id.
        sx={{ flexShrink: 0 }}
        label={
          <Stack direction="row" alignItems="center" gap={0.5}>
            <LockOutlinedIcon fontSize="small" />
            {label}
          </Stack>
        }
      />
    </Tooltip>
  );
};

export default BlockedChip;
