import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { format } from 'date-fns';
import { SEVERITY, TIME_FORMAT } from 'lib/constants';
import { type MRT_ColumnDef } from 'material-react-table';
import { AdvisorRun } from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import { Messages } from './AdvisorRuns.messages';
import { TRIGGERED_BY_LABEL } from '../insights/AdvisorInsights.utils';
import { formatDuration, isRunning } from './AdvisorRuns.utils';

const EM_DASH = '—';

const SEVERITY_CHIP_COLOR: Record<Severity, 'error' | 'warning' | 'info'> = {
  [Severity.emergency]: 'error',
  [Severity.alert]: 'error',
  [Severity.critical]: 'error',
  [Severity.error]: 'error',
  [Severity.warning]: 'warning',
  [Severity.notice]: 'info',
  [Severity.info]: 'info',
  [Severity.debug]: 'info',
  [Severity.unspecified]: 'info',
};

export const getRunsColumns = (): MRT_ColumnDef<AdvisorRun>[] => [
  {
    id: 'startedAt',
    header: Messages.columns.startedAt,
    accessorKey: 'startedAt',
    size: 170,
    grow: false,
    Cell: ({ row }) => (
      <Box component="span" sx={{ fontSize: '0.85rem' }}>
        {format(new Date(row.original.startedAt), TIME_FORMAT)}
      </Box>
    ),
  },
  {
    id: 'duration',
    header: Messages.columns.duration,
    enableSorting: false,
    size: 110,
    grow: false,
    Cell: ({ row }) =>
      isRunning(row.original) ? (
        <Stack direction="row" alignItems="center" gap={0.75}>
          <CircularProgress size={12} data-testid="run-in-progress" />
          <Typography variant="body2">{Messages.running}</Typography>
        </Stack>
      ) : (
        <span>{formatDuration(row.original) ?? EM_DASH}</span>
      ),
  },
  {
    id: 'triggeredBy',
    header: Messages.columns.triggeredBy,
    accessorFn: (row) => TRIGGERED_BY_LABEL[row.triggeredBy] || EM_DASH,
    size: 120,
    grow: false,
  },
  {
    id: 'findingsCount',
    header: Messages.columns.findings,
    accessorKey: 'findingsCount',
    size: 100,
    grow: false,
    Header: () => (
      <Tooltip title={Messages.tooltips.findings} arrow>
        <span>{Messages.columns.findings}</span>
      </Tooltip>
    ),
  },
  {
    id: 'severityCounts',
    header: Messages.columns.severity,
    enableSorting: false,
    size: 200,
    grow: true,
    Cell: ({ row }) => {
      const counts = row.original.severityCounts ?? [];
      if (!counts.length) {
        return <span>{EM_DASH}</span>;
      }
      return (
        <Stack direction="row" flexWrap="wrap" gap={0.5}>
          {counts.map(({ severity, count }) => (
            <Chip
              key={severity}
              size="small"
              color={SEVERITY_CHIP_COLOR[severity]}
              label={`${count} ${SEVERITY[severity]}`}
            />
          ))}
        </Stack>
      );
    },
  },
  {
    id: 'errorsCount',
    header: Messages.columns.errors,
    accessorKey: 'errorsCount',
    size: 120,
    grow: false,
    Header: () => (
      <Tooltip title={Messages.tooltips.errors} arrow>
        <span>{Messages.columns.errors}</span>
      </Tooltip>
    ),
  },
  {
    id: 'checksCount',
    header: Messages.columns.checks,
    accessorKey: 'checksCount',
    size: 90,
    grow: false,
  },
  {
    id: 'servicesCount',
    header: Messages.columns.services,
    accessorKey: 'servicesCount',
    size: 100,
    grow: false,
  },
];
