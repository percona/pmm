import { Chip, MRT_ColumnDef } from '@percona/percona-ui';
import { AlertsTableRow } from '../AlertsPage.types';
import { Stack, Typography } from '@mui/material';
import { STATUS_COLOR_MAP, STATUS_LABEL_MAP } from '../AlertsPage.constants';
import TriggeredAtCell from './TriggeredAtCell';

export const ALERT_STATUS_COLUMNS: MRT_ColumnDef<AlertsTableRow>[] = [
  {
    accessorKey: 'state',
    header: 'State',
    // State is filtered via the dropdown in the top toolbar.
    enableColumnFilter: false,
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
    id: 'activeAt',
    accessorFn: (row) =>
      row.type === 'alert' && row.activeAt ? new Date(row.activeAt) : undefined,
    header: 'Triggered at',
    filterVariant: 'datetime-range',
    filterFn: 'triggeredAtRangeFilterFn',
    sortingFn: 'datetime',
    Cell: ({ row: { original } }) =>
      original.type === 'alert' ? (
        <TriggeredAtCell activeAt={original.activeAt} />
      ) : null,
  },
];
