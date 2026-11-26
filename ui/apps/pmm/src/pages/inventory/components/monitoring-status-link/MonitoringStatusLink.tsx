import Link from '@mui/material/Link';
import type { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { getAgentsUrl } from '../../Inventory.utils';
import { type InventoryTab, MonitoringStatus } from '../../Inventory.types';

const STATUS_COLOR: Record<MonitoringStatus, string> = {
  [MonitoringStatus.ok]: 'success.main',
  [MonitoringStatus.warning]: 'warning.main',
  [MonitoringStatus.failed]: 'error.main',
};

interface Props {
  status: MonitoringStatus;
  tab: InventoryTab;
  id: string;
}

export const MonitoringStatusLink: FC<Props> = ({ status, tab, id }) => (
  <Link
    component={RouterLink}
    to={getAgentsUrl(tab, id)}
    color={STATUS_COLOR[status]}
    data-testid="monitoring-status-link"
  >
    {status}
  </Link>
);
