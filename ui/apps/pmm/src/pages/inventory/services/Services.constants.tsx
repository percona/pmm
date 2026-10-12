import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import { type MRT_ColumnDef } from 'material-react-table';
import { Link as RouterLink } from 'react-router-dom';
import { MonitoringStatusLink } from '../components/monitoring-status-link';
import { ServiceName } from '../components/service-name';
import { StatusChip } from '../components/status-chip';
import { StatusHeader } from '../components/status-header';
import {
  MONITORING_STATUS_OPTIONS,
  SERVICE_STATUS_OPTIONS,
} from '../Inventory.constants';
import { Messages as InventoryMessages } from '../Inventory.messages';
import { getFilteredTabUrl } from '../Inventory.utils';
import { Messages } from './Services.messages';
import type { ServiceRow, ServicesTableRow } from './Services.types';
import { isClusterRow } from './Services.utils';

// Cluster rows hold no values, so they only match filters through their services
const serviceValue =
  <K extends keyof ServiceRow>(key: K) =>
  (row: ServicesTableRow) =>
    isClusterRow(row) ? undefined : row[key];

export const SERVICES_COLUMNS: MRT_ColumnDef<ServicesTableRow>[] = [
  {
    id: 'status',
    size: 120,
    header: InventoryMessages.columns.status,
    Header: () => <StatusHeader />,
    accessorFn: serviceValue('status'),
    Cell: ({ row: { original } }) =>
      isClusterRow(original) ? null : <StatusChip status={original.status} />,
    filterVariant: 'select',
    filterSelectOptions: SERVICE_STATUS_OPTIONS,
    filterFn: 'equals',
    enableGlobalFilter: false,
  },
  {
    id: 'serviceName',
    size: 200,
    header: Messages.columns.serviceName,
    accessorFn: serviceValue('serviceName'),
    filterFn: 'contains',
    Cell: ({ row: { original } }) =>
      isClusterRow(original) ? (
        <Typography component="div" fontWeight="bold">
          <ServiceName
            name={original.name}
            serviceType={original.serviceType}
          />
        </Typography>
      ) : (
        <ServiceName
          name={original.serviceName}
          serviceType={original.serviceType}
        />
      ),
  },
  {
    id: 'nodeName',
    size: 180,
    header: Messages.columns.nodeName,
    accessorFn: serviceValue('nodeName'),
    filterFn: 'contains',
    Cell: ({ row: { original } }) =>
      isClusterRow(original) ? null : (
        <Link
          component={RouterLink}
          to={getFilteredTabUrl('nodes', 'nodeId', original.nodeId)}
        >
          {original.nodeName}
        </Link>
      ),
  },
  {
    id: 'agentsStatus',
    size: 140,
    header: InventoryMessages.columns.monitoring,
    accessorFn: serviceValue('agentsStatus'),
    Cell: ({ row: { original } }) =>
      isClusterRow(original) ? null : (
        <MonitoringStatusLink
          status={original.agentsStatus}
          tab="services"
          id={original.serviceId}
        />
      ),
    filterVariant: 'select',
    filterSelectOptions: MONITORING_STATUS_OPTIONS,
    filterFn: 'equals',
    enableGlobalFilter: false,
  },
  {
    id: 'address',
    size: 140,
    header: InventoryMessages.columns.address,
    accessorFn: serviceValue('address'),
    filterFn: 'contains',
  },
  {
    id: 'port',
    size: 80,
    header: Messages.columns.port,
    accessorFn: serviceValue('port'),
    filterFn: 'contains',
  },
  {
    id: 'serviceId',
    header: Messages.columns.serviceId,
    accessorFn: serviceValue('serviceId'),
    filterFn: 'contains',
  },
  // Never shown, it makes the cluster names searchable
  {
    id: 'cluster',
    header: Messages.columns.cluster,
    accessorFn: serviceValue('cluster'),
    filterFn: 'contains',
    enableHiding: false,
    visibleInShowHideMenu: false,
  },
];
