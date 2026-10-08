/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { useMemo } from 'react';
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Link,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { EmptyState } from './components/EmptyState';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import { Table, type MRT_ColumnDef } from '@percona/percona-ui';
import {
  useIsEstateRefreshing,
  useOmInventoryRuns,
  useRefreshInventory,
} from './inventoryHooks';
import { isRunActive } from './api';
import { useScanConflict } from './ScanFeedback';
import { RunStatusBadge } from './components/HealthBadge';
import { RunEntities } from './components/RunEntities';
import {
  formatCompactDuration,
  formatRunDuration,
  formatTimestamp,
  pluralize,
  runDurationSeconds,
} from './format';
import {
  ageSeconds,
  DEFAULT_RUN_LIMIT,
  groupRuns,
  isBoundedPeriod,
  isRunPeriod,
  RUN_PERIODS,
  WINDOWED_RUN_LIMIT,
  type OmRunGroup,
  type OmRunPeriod,
} from './inventory';
import { OM_ROUTE_NODES } from './constants';
import { useOmBase } from './useOmBase';
import type { OmInventoryRun, OmInventoryRunFailingNode } from './types';

/** When a group of scans ran: the newest, and for several, how many over how long. */
const GroupStarted = ({ group }: { group: OmRunGroup }) => {
  const newest = group.runs[0];
  if (group.runs.length === 1) {
    return <>{formatTimestamp(newest.start_time)}</>;
  }
  const oldest = group.runs[group.runs.length - 1];
  const span =
    (Date.parse(newest.start_time) - Date.parse(oldest.start_time)) / 1000;
  return (
    <Stack spacing={0.25}>
      <span>{formatTimestamp(newest.start_time)}</span>
      <Tooltip
        title={`From ${formatTimestamp(oldest.start_time)} to ${formatTimestamp(newest.start_time)}`}
      >
        <Typography
          variant="caption"
          color="text.secondary"
          data-testid="om-run-group-span"
        >
          {group.runs.length} scans over {formatCompactDuration(span) || '0s'}
        </Typography>
      </Tooltip>
    </Stack>
  );
};

/** Each node a scan failed on, linked to its row on the Nodes page. */
const FailingNodes = ({ nodes }: { nodes: OmInventoryRunFailingNode[] }) => {
  const omBase = useOmBase();
  if (nodes.length === 0) {
    return (
      <Box component="span" sx={{ color: 'text.disabled' }}>
        none
      </Box>
    );
  }
  return (
    <Stack direction="row" gap={1} flexWrap="wrap">
      {nodes.map((node) =>
        node.name ? (
          <Link
            key={node.node_id}
            component={RouterLink}
            to={`${omBase}/${OM_ROUTE_NODES}?node=${encodeURIComponent(node.name)}`}
          >
            {node.name}
          </Link>
        ) : (
          <span key={node.node_id}>{node.node_id}</span>
        )
      )}
    </Stack>
  );
};

/**
 * The scans a group stands for, each still openable to its own receipt.
 *
 * Unmounted while folded, so a group of a hundred scans does not ask for a hundred
 * receipts.
 */
const GroupRuns = ({
  group,
  expandRunId,
}: {
  group: OmRunGroup;
  expandRunId: string | null;
}) => (
  <Stack spacing={1} data-testid="om-run-group">
    {group.runs.map((run) => (
      <Accordion
        key={run.run_id}
        disableGutters
        variant="outlined"
        defaultExpanded={run.run_id === expandRunId}
        slotProps={{ transition: { unmountOnExit: true } }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="body2">
            {formatTimestamp(run.start_time)}, took{' '}
            {formatRunDuration(run.start_time, run.end_time) || '—'},{' '}
            {run.counts.answered_hosts} of {run.counts.probeable_hosts}{' '}
            {pluralize(run.counts.probeable_hosts, 'node')} answered
          </Typography>
        </AccordionSummary>
        <AccordionDetails>
          <RunEntities run={run} />
        </AccordionDetails>
      </Accordion>
    ))}
  </Stack>
);

const RUN_COLUMNS: MRT_ColumnDef<OmRunGroup>[] = [
  {
    accessorFn: (group) => group.runs[0].status,
    id: 'status',
    header: 'Status',
    Cell: ({ row: { original } }) => (
      <RunStatusBadge status={original.runs[0].status} />
    ),
  },
  {
    // A group of scans is placed by its newest, and says how many and over how long:
    // "it has failed like this for six hours" is what grouping exists to show.
    accessorFn: (group) => group.runs[0].start_time,
    id: 'start_time',
    header: 'Started',
    Cell: ({ row: { original } }) => <GroupStarted group={original} />,
  },
  {
    // Sorts on elapsed seconds, not the formatted string -- lexicographically "9s"
    // lands after "10m". NodesPage's age column already does it this way.
    accessorFn: (group) =>
      runDurationSeconds(group.runs[0].start_time, group.runs[0].end_time),
    id: 'duration',
    header: 'Duration',
    Cell: ({ row: { original } }) =>
      formatRunDuration(
        original.runs[0].start_time,
        original.runs[0].end_time
      ) || '—',
  },
  {
    // Empty rather than zero for a full sweep: "the whole estate" is the ordinary
    // case, and a column that said "all" on nineteen rows out of twenty would be
    // noise. What matters is spotting the scoped one among them.
    accessorFn: (group) => group.runs[0].scope.length,
    id: 'scope',
    header: 'Scope',
    Cell: ({ row: { original } }) =>
      original.runs[0].scope.length === 0 ? (
        <Tooltip title="Every node">
          <Box component="span" sx={{ color: 'text.disabled' }}>
            all
          </Box>
        </Tooltip>
      ) : (
        <Tooltip title={original.runs[0].scope.join(', ')}>
          <Box component="span">
            {original.runs[0].scope.length} host
            {original.runs[0].scope.length === 1 ? '' : 's'}
          </Box>
        </Tooltip>
      ),
  },
  {
    accessorFn: (group) => group.runs[0].counts.answered_hosts,
    id: 'hosts',
    header: 'Nodes',
    Cell: ({ row: { original } }) => (
      // total / probeable / answered in one cell. The gap between the first two is
      // the estate nothing can be dispatched to, which is an onboarding fact rather
      // than a failed run, and the gap between the last two is what actually failed.
      <Tooltip
        title={`${original.runs[0].counts.total_hosts} in scope, ${original.runs[0].counts.probeable_hosts} reachable, ${original.runs[0].counts.answered_hosts} answered`}
      >
        <Box component="span">
          {original.runs[0].counts.answered_hosts}/
          {original.runs[0].counts.probeable_hosts}
          {original.runs[0].counts.probeable_hosts ===
          original.runs[0].counts.total_hosts
            ? ''
            : ` of ${original.runs[0].counts.total_hosts}`}
        </Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (group) => group.runs[0].counts.total_services,
    id: 'services',
    header: 'Services',
    Cell: ({ row: { original } }) => original.runs[0].counts.total_services,
  },
  {
    accessorFn: (group) => group.runs[0].counts.resolved_services,
    id: 'resolved',
    header: 'Resolved',
    Cell: ({ row: { original } }) => (
      <Tooltip title="Services on a node with a working automation agent">
        <Box component="span">{original.runs[0].counts.resolved_services}</Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (group) => group.runs[0].counts.answered_services,
    id: 'answered',
    header: 'Answered',
    Cell: ({ row: { original } }) => (
      // The diagnostic pair: resolved says the mapping worked, answered says the
      // node ran the payload. resolved=9 / answered=0 is a healthy mapping and
      // broken executors — a distinction a single "failed" count would hide.
      <Tooltip title="Services whose node ran the scan">
        <Box component="span">{original.runs[0].counts.answered_services}</Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (group) => group.runs[0].counts.orphaned_services,
    id: 'orphaned',
    header: 'Orphaned',
    Cell: ({ row: { original } }) => (
      <Tooltip title="Services with no automation agent — not an error">
        <Box component="span">{original.runs[0].counts.orphaned_services}</Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (group) => (group.runs[0].failing_nodes ?? []).length,
    id: 'failing',
    header: 'Failing nodes',
    Cell: ({ row: { original } }) => (
      <FailingNodes nodes={original.runs[0].failing_nodes ?? []} />
    ),
  },
];

/**
 * Ask for a refresh, and say honestly what came back.
 *
 * Separate from `SyncButton`, which rebuilds pmm-managed's snapshot in about a tenth
 * of a second and never touches a host. This queues a job per host and takes tens of
 * seconds, so the app answers `running` and the list below polls until it lands. Two
 * triggers that different must not look alike, which is most of why they live on
 * different pages.
 */
const RefreshButton = () => {
  const trigger = useRefreshInventory();
  // Any active run, not just the newest. Refreshes are host-scoped, so two can overlap
  // and a narrow one started later can reach a terminal status while a broader one is
  // still probing - `runs[0]` would re-enable this button into a sweep that must
  // conflict with it. Same reason `useOmInventoryRuns` polls on the whole collection.
  const running = useIsEstateRefreshing();
  // A 409 is an expected answer rather than a failure, and since conflict is judged
  // per host the message names what is in flight instead of saying "a sweep is
  // already running" - which was true of anything and useful for nothing.
  const { conflict } = useScanConflict(trigger);
  const failure = trigger.error && !conflict ? trigger.error : null;

  return (
    <Stack direction="row" alignItems="center" gap={1}>
      <Tooltip title="Scan every node now, collecting what no metric carries">
        <span>
          <Button
            variant="contained"
            startIcon={
              running ? <CircularProgress size={16} /> : <PlayArrowIcon />
            }
            disabled={running || trigger.isPending}
            onClick={() => trigger.refreshAll()}
          >
            {running ? 'Scanning…' : 'Scan all nodes'}
          </Button>
        </span>
      </Tooltip>
      {conflict && (
        <Typography variant="body2" color="text.secondary">
          {conflict.message}
        </Typography>
      )}
      {failure && (
        <Typography variant="body2" color="error">
          Could not start a scan: {failure.message}
        </Typography>
      )}
    </Stack>
  );
};

/**
 * What OM currently knows, from the newest run.
 *
 * The table answers "has this been working". This answers "what does OM know right
 * now", which is the more common question and otherwise needs reading the first row
 * of a table and knowing that the first row is the newest.
 *
 * A run stuck in `running` is a real state with a reaper behind it, so its age is
 * shown: that is what distinguishes a refresh that is working from one that is
 * wedged.
 */
const LastRun = ({ run }: { run: OmInventoryRun | undefined }) => {
  if (!run) {
    return (
      <EmptyState title="No scans yet">
        Operations scans your nodes on a schedule to collect what no metric
        carries - the installed version, the command line, the config file.
        Nothing has run yet, so there is nothing to show.
      </EmptyState>
    );
  }
  const age = ageSeconds(run.start_time);
  const active = isRunActive(run.status);
  return (
    <Stack
      direction="row"
      gap={3}
      alignItems="center"
      sx={{ flexWrap: 'wrap' }}
    >
      <RunStatusBadge status={run.status} />
      <Typography variant="body2" color="text.secondary">
        {active
          ? `started ${age == null ? 'just now' : `${formatCompactDuration(age)} ago`}`
          : `${age == null ? '' : `${formatCompactDuration(age)} ago`}, took ${
              formatRunDuration(run.start_time, run.end_time) || '—'
            }`}
      </Typography>
      <Typography variant="body2">
        <strong>{run.counts.answered_hosts}</strong> of{' '}
        {run.counts.probeable_hosts}{' '}
        {pluralize(run.counts.probeable_hosts, 'node')} answered
      </Typography>
      <Typography variant="body2">
        <strong>{run.counts.answered_services}</strong> of{' '}
        {run.counts.resolved_services}{' '}
        {pluralize(run.counts.resolved_services, 'service')} answered
      </Typography>
      {run.scope.length > 0 && (
        <Tooltip title={run.scope.join(', ')}>
          <Typography variant="body2" color="warning.main">
            scoped to {run.scope.length} host
            {run.scope.length === 1 ? '' : 's'}
          </Typography>
        </Tooltip>
      )}
      {run.error && (
        <Typography variant="body2" color="error">
          {run.error}
        </Typography>
      )}
    </Stack>
  );
};

/**
 * One chip per `RUN_PERIODS` entry, in that order — adding a quick filter there is
 * the whole change; nothing here names a period.
 */
const PeriodFilter = ({
  value,
  onChange,
}: {
  value: OmRunPeriod;
  onChange: (next: OmRunPeriod) => void;
}) => (
  <Stack direction="row" gap={1} flexWrap="wrap">
    {RUN_PERIODS.map((option) => (
      <Chip
        key={option.id}
        size="small"
        label={option.label}
        color="default"
        variant={value === option.id ? 'filled' : 'outlined'}
        onClick={() => onChange(option.id)}
      />
    ))}
  </Stack>
);

/**
 * The scan history: every pass Operations made over the nodes, and what each found.
 *
 * These are the app's scans, not pmm-managed's collection pass: one runs a payload on
 * every node and takes tens of seconds, the other recomputes a document from data PMM
 * already holds. Two different things once both called a "run", which is why the fleet
 * reading and this one are different pages.
 *
 * Read through pmm-managed rather than from PMM Extensions directly, which is what lets this tab
 * render its own error when PMM Extensions is unwell instead of being blanked by a gate that
 * fails closed.
 *
 * A tab on Automations, beside installs: both answer "what has run, and did it work",
 * and the page that used to hold this one was called Inventory, which collided with
 * PMM's own and told a reader nothing about what was on it.
 */
export const AutomationsScansTab = () => {
  // In the query string rather than component state, so a link to a window is
  // shareable and a reload does not silently put the reader back on the default.
  const [params, setParams] = useSearchParams();
  // Same reason the tab lives in the URL: a link to "last month" should open last
  // month. Default is the week window — All is still the uncapped-history view, and
  // that is the one that becomes unreadable.
  const requestedPeriod = params.get('period');
  const period: OmRunPeriod = isRunPeriod(requestedPeriod)
    ? requestedPeriod
    : 'week';
  // Newest overall, unfiltered: "what does OM know right now" is not a claim about
  // the selected window, and an empty last-week table must not read as "no refresh
  // has ever run".
  const latest = useOmInventoryRuns();
  // `since` is not computed here: the hook derives it from `period` fresh on every
  // fetch, so a chip left open keeps meaning what it says instead of the window
  // quietly growing past its own label. See OmRunFilters in inventoryHooks.ts.
  const {
    data: runs,
    isLoading,
    error,
  } = useOmInventoryRuns({
    period,
    limit: isBoundedPeriod(period) ? WINDOWED_RUN_LIMIT : DEFAULT_RUN_LIMIT,
  });
  // Consecutive scans that ended the same way are one row: a node broken since
  // Tuesday reads as one row, not one per scan.
  const rows = useMemo(() => groupRuns(runs ?? []), [runs]);
  // `?expand=<run_id>` unfolds that scan's row on landing - the group holding it, and
  // the scan itself inside the group.
  const expandRunId = params.get('expand');
  const expandGroupId = rows.find((group) =>
    group.runs.some((run) => run.run_id === expandRunId)
  )?.id;

  const setPeriod = (next: OmRunPeriod) =>
    setParams((current) => {
      const updated = new URLSearchParams(current);
      updated.set('period', next);
      return updated;
    });

  return (
    <Stack gap={2}>
      {/* The scan-specific controls, in the tab rather than the page header: a period
          filter over installs would mean nothing, and this action scans the nodes
          rather than starting an install. */}
      <Stack
        direction="row"
        alignItems="center"
        gap={2}
        justifyContent="flex-end"
      >
        <PeriodFilter value={period} onChange={setPeriod} />
        <RefreshButton />
      </Stack>

      {error && (
        <Alert severity="error">
          {/* Rendered inside the page rather than replacing it: PMM Extensions being
                  unwell is a fact about the fleet, and the installs tab still
                  reads. */}
          Could not load scans: {(error as Error).message}
        </Alert>
      )}

      {/* Only once the query has actually answered. LastRun reads an absent run as
              "no scan has run yet", which is a claim about the fleet - not something
              to assert while the first request is still in flight or has failed with no
              cached rows to fall back on. */}
      {(!latest.isLoading || latest.data) && !latest.error && (
        <LastRun run={latest.data?.[0]} />
      )}

      {isLoading && !runs ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
          <CircularProgress />
        </Box>
      ) : rows.length === 0 && !error ? (
        <EmptyState title="No scans in this period">
          Scans run on a schedule, so a short window can be empty while
          everything is working. Widen the period, or scan now.
        </EmptyState>
      ) : (
        <Table
          tableName="om-inventory-runs"
          columns={RUN_COLUMNS}
          data={rows}
          getRowId={(group) => group.id}
          enableGlobalFilter={false}
          enableColumnFilters={false}
          enableHiding={false}
          enablePagination={false}
          enableStickyHeader
          enableExpanding
          initialState={
            expandGroupId ? { expanded: { [expandGroupId]: true } } : undefined
          }
          renderDetailPanel={({ row }) =>
            row.original.runs.length === 1 ? (
              <RunEntities run={row.original.runs[0]} />
            ) : (
              <GroupRuns group={row.original} expandRunId={expandRunId} />
            )
          }
        />
      )}
    </Stack>
  );
};
