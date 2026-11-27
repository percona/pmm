import { Chip, Tooltip } from '@percona/peak-ui';
import type { FC } from 'react';
import { Messages } from './TruncatedChip.messages';

export interface Props {
  dataTestId?: string;
}

// MySQL leaves no marker of its own on statement text it cut short.
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
