import { Chip, MRT_ColumnDef } from '@percona/percona-ui';
import { AlertsTableRow } from '../AlertsPage.types';
import type { AlertSeverity as Severity } from 'types/alerting.types';
import { Stack, Typography } from '@mui/material';
import NotificationsOffOutlinedIcon from '@mui/icons-material/NotificationsOffOutlined';
import { STATUS_COLOR_MAP, STATUS_LABEL_MAP } from '../AlertsPage.constants';
import { Messages } from './AlertStatusTable.messages';
import TriggeredAtCell from './TriggeredAtCell';
import { AlertSeverity } from '../alert-severity';
import {
  sortByState,
  sortByNode,
  sortByService,
} from './AlertStatusTable.utils';

const ALERT_SEVERITY_OPTIONS = [
  { value: 'emergency', label: 'Emergency' },
  { value: 'alert', label: 'Alert' },
  { value: 'critical', label: 'Critical' },
  { value: 'error', label: 'Error' },
  { value: 'warning', label: 'Warning' },
  { value: 'notice', label: 'Notice' },
  { value: 'info', label: 'Info' },
  { value: 'debug', label: 'Debug' },
  { value: 'unknown', label: 'Unknown' },
];

export const ALERT_STATUS_COLUMNS: MRT_ColumnDef<AlertsTableRow>[] = [
  {
    accessorKey: 'state',
    header: 'State',
    // State is filtered via the dropdown in the top toolbar.
    enableColumnFilter: false,
    sortingFn: (a, b) => sortByState(a.original, b.original),
    Cell: ({ row: { original } }) => {
      if (original.type === 'alert') {
        return (
          <Stack direction="row" alignItems="center" gap={1}>
            {original.silenced ? (
              <Chip
                color="info"
                label={
                  <Stack direction="row" alignItems="center" gap={0.5}>
                    <NotificationsOffOutlinedIcon fontSize="small" />
                    {Messages.silenced}
                  </Stack>
                }
              />
            ) : (
              <Chip
                label={STATUS_LABEL_MAP[original.state]}
                color={STATUS_COLOR_MAP[original.state]}
              />
            )}
            <Typography noWrap>
              for {original.silenced ? original.silencedAge : original.age}
            </Typography>
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
    sortingFn: (a, b) => sortByNode(a.original, b.original),
    Cell: ({ row: { original } }) => (
      <Typography fontWeight={original.type === 'node' ? 'bold' : undefined}>
        {original.nodeId}
      </Typography>
    ),
  },
  {
    accessorKey: 'serviceName',
    header: 'Service',
    sortingFn: (a, b) => sortByService(a.original, b.original),
  },
  {
    id: 'severity',
    accessorFn: (row) => row.type === 'alert' && row.labels.severity,
    header: 'Severity',
    filterVariant: 'multi-select',
    filterSelectOptions: ALERT_SEVERITY_OPTIONS,
    Cell: ({ row: { original } }) => {
      if (original.type === 'alert') {
        return (
          <AlertSeverity severity={original.labels.severity as Severity} />
        );
      }

      return null;
    },
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
