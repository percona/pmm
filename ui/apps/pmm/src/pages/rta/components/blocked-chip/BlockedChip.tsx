import Stack from '@mui/material/Stack';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { Chip, Tooltip } from '@percona/peak-ui';
import { FC } from 'react';
import { BlockingTransaction } from 'types/rta.types';
import { soleBlockerOf } from 'pages/rta/overview/table/OverviewTable.utils';
import { Messages } from './BlockedChip.messages';

export interface Props {
  // Every transaction holding this statement up. The chip names one only when one is
  // actually the answer; otherwise it reports the count, because claiming a single culprit
  // that is not the whole story sends the reader after the wrong connection.
  blockers: BlockingTransaction[];
}

const BlockedChip: FC<Props> = ({ blockers }) => {
  const sole = soleBlockerOf(blockers);

  return (
    <Tooltip
      title={
        sole
          ? Messages.tooltip(sole.blockingConnId)
          : Messages.tooltipMany(blockers.length)
      }
      arrow
    >
      <Chip
        color="warning"
        data-testid="blocked-chip"
        // Never shrink: this sits beside a width:100% query cell, and letting the browser
        // squeeze it clips the connection id the chip exists to carry.
        sx={{ flexShrink: 0 }}
        label={
          <Stack direction="row" alignItems="center" gap={0.5}>
            <LockOutlinedIcon fontSize="small" />
            {sole
              ? Messages.blockedBy(sole.blockingConnId)
              : Messages.blockedByMany(blockers.length)}
          </Stack>
        }
      />
    </Tooltip>
  );
};

export default BlockedChip;
