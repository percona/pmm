import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { format } from 'date-fns';
import { ADVISOR_INTERVAL, SEVERITY, TIME_FORMAT } from 'lib/constants';
import { type MRT_ColumnDef } from 'material-react-table';
import { type FC } from 'react';
import { type AdvisorRun, AdvisorRunStatus } from 'types/advisors.types';
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

interface CoverageProps {
  run: AdvisorRun;
  done: number;
  planned: number;
  testId: string;
}

// what the run covered out of what it planned, as "n/N"; a run without a plan
// (queued, aborted, or stopped while planning) shows a dash
const Coverage: FC<CoverageProps> = ({ run, done, planned, testId }) => {
  const completed = run.status === AdvisorRunStatus.completed;
  if (!planned && !completed) {
    return <span data-testid={testId}>{EM_DASH}</span>;
  }

  // a finished run that fell short may be worth running again
  const short = !isRunning(run) && done < planned;
  return (
    <Tooltip title={planned ? '' : Messages.tooltips.nothingToCheck} arrow>
      <span data-testid={testId}>
        <Box
          component="span"
          sx={short ? { color: 'warning.main' } : undefined}
          data-short={short ? 'true' : undefined}
        >
          {done}
        </Box>
        /{planned}
      </span>
    </Tooltip>
  );
};

export const getRunsColumns = (): MRT_ColumnDef<AdvisorRun>[] => [
  {
    id: 'startedAt',
    header: Messages.columns.startedAt,
    accessorKey: 'startedAt',
    size: 165,
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
    size: 100,
    grow: false,
    Cell: ({ row }) => {
      switch (row.original.status) {
        case AdvisorRunStatus.queued:
        case AdvisorRunStatus.running: {
          const queued = row.original.status === AdvisorRunStatus.queued;
          return (
            <Tooltip title={queued ? Messages.tooltips.queued : ''} arrow>
              <Stack direction="row" alignItems="center" gap={0.75}>
                <CircularProgress size={12} data-testid="run-in-progress" />
                <Typography variant="body2">
                  {queued ? Messages.queued : Messages.running}
                </Typography>
              </Stack>
            </Tooltip>
          );
        }
        case AdvisorRunStatus.interrupted:
        case AdvisorRunStatus.aborted: {
          const interrupted =
            row.original.status === AdvisorRunStatus.interrupted;
          return (
            <Tooltip
              title={
                interrupted
                  ? Messages.tooltips.interrupted
                  : Messages.tooltips.aborted
              }
              arrow
            >
              <Typography
                variant="body2"
                color={interrupted ? 'warning.main' : 'error.main'}
                data-testid={interrupted ? 'run-interrupted' : 'run-aborted'}
              >
                {interrupted ? Messages.interrupted : Messages.aborted}
              </Typography>
            </Tooltip>
          );
        }
        default:
          return <span>{formatDuration(row.original) ?? EM_DASH}</span>;
      }
    },
  },
  {
    id: 'triggeredBy',
    header: Messages.columns.triggeredBy,
    // the interval groups a run was narrowed to, e.g. "Scheduler · Frequent"
    accessorFn: (row) =>
      [
        TRIGGERED_BY_LABEL[row.triggeredBy] || EM_DASH,
        (row.intervals ?? []).map((i) => ADVISOR_INTERVAL[i]).join(', '),
      ]
        .filter(Boolean)
        .join(' · '),
    size: 150,
    grow: false,
  },
  {
    id: 'checksCount',
    header: Messages.columns.checks,
    size: 90,
    grow: false,
    Header: () => (
      <Tooltip title={Messages.tooltips.checks} arrow>
        <span>{Messages.columns.checks}</span>
      </Tooltip>
    ),
    Cell: ({ row }) => (
      <Coverage
        run={row.original}
        done={row.original.checksCount}
        planned={row.original.plannedChecksCount}
        testId="run-checks"
      />
    ),
  },
  {
    id: 'findingsCount',
    header: Messages.columns.findings,
    accessorKey: 'findingsCount',
    size: 95,
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
    size: 150,
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
    size: 85,
    grow: false,
    Header: () => (
      <Tooltip title={Messages.tooltips.errors} arrow>
        <span>{Messages.columns.errors}</span>
      </Tooltip>
    ),
  },
  {
    id: 'servicesCount',
    header: Messages.columns.services,
    size: 95,
    grow: false,
    Header: () => (
      <Tooltip title={Messages.tooltips.services} arrow>
        <span>{Messages.columns.services}</span>
      </Tooltip>
    ),
    Cell: ({ row }) => (
      <Coverage
        run={row.original}
        done={row.original.servicesCount}
        planned={row.original.plannedServicesCount}
        testId="run-services"
      />
    ),
  },
];
