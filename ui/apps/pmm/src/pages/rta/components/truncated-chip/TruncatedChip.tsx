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

// MySQL and PostgreSQL leave no marker of their own on statement text they cut short.
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
