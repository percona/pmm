import { Chip } from '@percona/peak-ui';
import type { FC } from 'react';
import type { ServiceStatus } from 'types/services.types';
import { getStatusColor, getStatusText } from '../../Inventory.utils';

interface Props {
  status: ServiceStatus;
}

export const StatusChip: FC<Props> = ({ status }) => (
  <Chip
    size="small"
    label={getStatusText(status)}
    color={getStatusColor(status)}
    data-testid="status-chip"
  />
);
