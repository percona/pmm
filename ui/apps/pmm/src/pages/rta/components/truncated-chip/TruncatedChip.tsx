import { Chip, Tooltip } from '@percona/peak-ui';
import type { FC } from 'react';
import { Messages } from './TruncatedChip.messages';

export interface Props {
  dataTestId?: string;
}

// Marks statement text that MySQL cut short. MySQL leaves no marker of its own, so
// without this a cut statement reads as a complete one.
const TruncatedChip: FC<Props> = ({ dataTestId = 'truncated-chip' }) => (
  <Tooltip title={Messages.tooltip} arrow>
    <Chip
      size="small"
      color="default"
      data-testid={dataTestId}
      // Never shrink: it sits beside a width:100% query cell.
      sx={{ flexShrink: 0 }}
      label={Messages.label}
    />
  </Tooltip>
);

export default TruncatedChip;
