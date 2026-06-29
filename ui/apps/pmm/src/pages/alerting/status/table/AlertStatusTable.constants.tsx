import { Chip, MRT_ColumnDef } from '@percona/percona-ui';
import { AlertsTableRow } from '../AlertsPage.types';
import { Stack, Typography } from '@mui/material';
import { formatTriggeredAt } from './AlertStatusTable.utils';
import { STATUS_COLOR_MAP, STATUS_LABEL_MAP } from '../AlertsPage.constants';

export const ALERT_STATUS_COLUMNS: MRT_ColumnDef<AlertsTableRow>[] = [
  {
    accessorKey: 'state',
    header: 'State',
    Cell: ({ row: { original } }) => {
      if (original.type === 'alert') {
        return (
          <Stack direction="row" alignItems="center" gap={1}>
            <Chip
              label={STATUS_LABEL_MAP[original.state]}
              color={STATUS_COLOR_MAP[original.state]}
            />
            <Typography>for {original.age}</Typography>
          </Stack>
        );
      }

      return null;
    },
  },
  {
    accessorKey: 'alertName',
    header: 'Name',
  },
  {
    accessorKey: 'nodeId',
    header: 'Node',
    Cell: ({ row: { original } }) => (
      <Typography fontWeight={original.type === 'node' ? 'bold' : undefined}>
        {original.nodeId}
      </Typography>
    ),
  },
  {
    accessorKey: 'serviceName',
    header: 'Service',
  },
  {
    accessorKey: 'activeAt',
    header: 'Triggered at',
    Cell: ({ row: { original } }) =>
      original.type === 'alert'
        ? formatTriggeredAt(original.activeAt, original.timezone)
        : null,
  },
];
