import { Chip, Tooltip } from '@percona/peak-ui';
import type { FC } from 'react';
import { ServiceType } from 'types/services.types';
import { Messages } from './TruncatedChip.messages';

export interface Props {
  dataTestId?: string;
  technology?: ServiceType;
}

const TOOLTIPS: Partial<Record<ServiceType, string>> = {
  [ServiceType.mysql]: Messages.tooltip,
  [ServiceType.posgresql]: Messages.tooltipPostgreSql,
};

// Marks statement text that MySQL or PostgreSQL cut short. Neither leaves a marker of its own, so
// without this a cut statement reads as a complete one.
const TruncatedChip: FC<Props> = ({
  dataTestId = 'truncated-chip',
  technology,
}) => (
  <Tooltip
    title={(technology && TOOLTIPS[technology]) || Messages.tooltip}
    arrow
  >
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
