import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import { Chip } from '@percona/peak-ui';
import { type MRT_ColumnDef } from 'material-react-table';
import { Link as RouterLink } from 'react-router-dom';
import { NodeType } from 'types/nodes.types';
import { MonitoringStatusLink } from '../components/monitoring-status-link';
import { StatusChip } from '../components/status-chip';
import {
  MONITORING_STATUS_OPTIONS,
  SERVICE_STATUS_OPTIONS,
} from '../Inventory.constants';
import { Messages as InventoryMessages } from '../Inventory.messages';
import { getFilteredTabUrl } from '../Inventory.utils';
import { Messages } from './Nodes.messages';
import type { NodeRow } from './Nodes.types';

const NODE_TYPE_OPTIONS = [
  { label: 'Container', value: NodeType.container },
  { label: 'Generic', value: NodeType.generic },
  { label: 'Remote', value: NodeType.remote },
  { label: 'RemoteAzureDB', value: NodeType.remoteAzureDatabase },
  { label: 'RemoteRDS', value: NodeType.remoteRds },
];

export const NODES_COLUMNS: MRT_ColumnDef<NodeRow>[] = [
  {
    accessorKey: 'status',
    size: 110,
    header: InventoryMessages.columns.status,
    Cell: ({ row: { original } }) => <StatusChip status={original.status} />,
    filterVariant: 'select',
    filterSelectOptions: SERVICE_STATUS_OPTIONS,
    filterFn: 'equals',
    enableGlobalFilter: false,
  },
  {
    accessorKey: 'nodeName',
    size: 190,
    header: Messages.columns.nodeName,
    filterFn: 'contains',
    Cell: ({ row: { original } }) => (
      <Stack direction="row" alignItems="center" gap={1}>
        <span>{original.nodeName}</span>
        {original.haRole && (
          <Chip
            size="small"
            label={Messages.haRole[original.haRole]}
            data-testid="ha-role-chip"
          />
        )}
      </Stack>
    ),
  },
  {
    accessorKey: 'nodeType',
    size: 130,
    header: Messages.columns.nodeType,
    filterVariant: 'select',
    filterSelectOptions: NODE_TYPE_OPTIONS,
    filterFn: 'equals',
    enableGlobalFilter: false,
  },
  {
    accessorKey: 'agentsStatus',
    size: 130,
    header: InventoryMessages.columns.monitoring,
    Cell: ({ row: { original } }) => (
      <MonitoringStatusLink
        status={original.agentsStatus}
        tab="nodes"
        id={original.nodeId}
      />
    ),
    filterVariant: 'select',
    filterSelectOptions: MONITORING_STATUS_OPTIONS,
    filterFn: 'equals',
    enableGlobalFilter: false,
  },
  {
    accessorKey: 'address',
    size: 140,
    header: InventoryMessages.columns.address,
    filterFn: 'contains',
  },
  {
    id: 'services',
    size: 170,
    header: Messages.columns.services,
    accessorFn: ({ services }) =>
      services.map(({ serviceName }) => serviceName).join(', '),
    filterFn: 'contains',
    Cell: ({ row: { original } }) => {
      const { services } = original;

      if (services.length === 1) {
        return (
          <Link
            component={RouterLink}
            to={getFilteredTabUrl(
              'services',
              'serviceId',
              services[0].serviceId
            )}
          >
            {services[0].serviceName}
          </Link>
        );
      }

      return services.length
        ? Messages.servicesCount(services.length)
        : Messages.noServices;
    },
  },
  {
    accessorKey: 'nodeId',
    header: Messages.columns.nodeId,
    filterFn: 'contains',
  },
  // Never shown, it is filtered on from the toolbar
  {
    id: 'isPmmServerNode',
    header: Messages.columns.isPmmServerNode,
    accessorFn: ({ isPmmServerNode }) => String(isPmmServerNode),
    filterFn: 'equals',
    enableGlobalFilter: false,
    enableHiding: false,
    visibleInShowHideMenu: false,
  },
];
