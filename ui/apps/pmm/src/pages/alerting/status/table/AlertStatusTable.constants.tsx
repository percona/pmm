import { Chip, MRT_ColumnDef } from '@percona/peak-ui';
import { AlertsTableRow } from '../AlertsPage.types';
import { Stack, Typography } from '@mui/material';
import NotificationsOffOutlinedIcon from '@mui/icons-material/NotificationsOffOutlined';
import { STATUS_COLOR_MAP, STATUS_LABEL_MAP } from '../AlertsPage.constants';
import { Messages } from './AlertStatusTable.messages';
import TriggeredAtCell from './TriggeredAtCell';
import {
  AlertSeverity,
  SEVERITY_BAND_CONFIG,
  SEVERITY_BAND_ORDER,
} from '../alert-severity';
import {
  sortByState,
  sortByNode,
  sortByService,
  sortBySeverity,
  getSeverityFilterValue,
  UNKNOWN_SEVERITY,
} from './AlertStatusTable.utils';

const ALERT_SEVERITY_OPTIONS = [
  ...SEVERITY_BAND_ORDER.map((band) => ({
    value: band,
    label: SEVERITY_BAND_CONFIG[band].label,
  })),
  { value: UNKNOWN_SEVERITY, label: 'Unknown' },
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
    accessorFn: (row) =>
      row.type === 'alert' && getSeverityFilterValue(row.labels.severity),
    header: 'Severity',
    filterVariant: 'multi-select',
    filterSelectOptions: ALERT_SEVERITY_OPTIONS,
    sortingFn: (a, b) => sortBySeverity(a.original, b.original),
    Cell: ({ row: { original } }) => {
      if (original.type === 'alert') {
        return <AlertSeverity severity={original.labels.severity} />;
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
