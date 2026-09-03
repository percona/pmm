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

import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  LinearProgress,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  MaterialReactTable,
  useMaterialReactTable,
  type MRT_ColumnDef,
} from 'material-react-table';
import {
  HOST_DATABASE_STATE_COLOR,
  HOST_DATABASE_STATE_LABEL,
  HOST_DATABASE_STATE_PHRASE,
} from './constants';
import { OmHeader } from './components/OmHeader';
import { Unavailable } from './components/Unavailable';
import { formatCompactDuration } from './format';
import { ageSeconds, isFailing, toHostRows } from './inventory';
import {
  useForgetHost,
  useIsEstateRefreshing,
  useOmInventoryHosts,
  useRefreshInventory,
  useTriggerHostBootstrap,
} from './inventoryHooks';
import { OmApiError } from './api';
import type { OmHostBootstrapAccepted, OmHostRow } from './types';

/** Identifiers and long text the table carries but does not open with. */
const HIDDEN_BY_DEFAULT = {
  node_id: false,
  address: false,
  kernel: false,
};

/**
 * The Hosts page's own filter, on top of what the estate returns.
 *
 * `all` is the default: a filter that hides rows by default reads as data loss the
 * first time it hides something a reader expected to see, which is exactly what
 * happened here in review — a host that already had a database on it vanished
 * under the old `available`-by-default and looked like a sync bug. `unmonitored`
 * and `monitored` are still one click away, for the two narrower questions
 * ("where could I install something" / "what is PMM already watching") — they
 * just no longer answer themselves on page load.
 */
type HostFilter = 'unmonitored' | 'monitored' | 'all';

const HOST_FILTERS: { id: HostFilter; label: string }[] = [
  { id: 'unmonitored', label: 'Not monitored' },
  { id: 'monitored', label: 'Monitored' },
  { id: 'all', label: 'All' },
];

/**
 * `unregistered_only` counts as not monitored: a host with a mongod PMM cannot see
 * is not a place a fresh install can safely target, but it is also not one PMM is
 * monitoring — grouping it with `has_service` would hide it from both filters.
 */
function matchesHostFilter(row: OmHostRow, filter: HostFilter): boolean {
  if (filter === 'all') {
    return true;
  }
  const monitored = row.database_state === 'has_service';
  return filter === 'monitored' ? monitored : !monitored;
}

const HostFilterChips = ({
  value,
  onChange,
}: {
  value: HostFilter;
  onChange: (next: HostFilter) => void;
}) => (
  <Stack direction="row" gap={1} flexWrap="wrap">
    {HOST_FILTERS.map((option) => (
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
 * Why nothing can run on a host, in the three ways it can be true.
 *
 * Kept as one cell rather than three columns because they are not independent: a host
 * that is not registered cannot be reachable, and reading three booleans to reach one
 * conclusion is work the page should do for its reader. The distinction that matters
 * is what to go and do, so that is what the cell says.
 */
const ExecutorCell = ({ row }: { row: OmHostRow }) => {
  const { registered, reachable, driver_healthy, detail } = row.executor;
  if (!registered) {
    return (
      <Tooltip title="No executor client is registered for this host, so nothing can be run on it. It has never been onboarded, or its registration was removed.">
        <Chip size="small" variant="outlined" label="Not onboarded" />
      </Tooltip>
    );
  }
  if (!reachable) {
    return (
      <Tooltip title="An executor client is registered for this host but the backend has lost contact with it. The machine is down, or its agent is stopped.">
        <Chip size="small" color="error" label="Agent down" />
      </Tooltip>
    );
  }
  if (!driver_healthy) {
    return (
      <Tooltip
        title={
          detail ??
          'The executor client is up but its raw_exec driver is unhealthy, so it cannot run a probe.'
        }
      >
        <Chip size="small" color="warning" label="Driver unhealthy" />
      </Tooltip>
    );
  }
  return <Chip size="small" color="success" variant="outlined" label="Ready" />;
};

/**
 * Whether OM automation can actually run something on this host, and why not
 * when it cannot.
 *
 * Deliberately separate from `ExecutorCell`, not a duplicate of it:
 * `automation_eligible` is `executor` plus `pmm_agent_connected`, a signal
 * `executor` alone cannot carry - a host can have a perfectly healthy Nomad
 * executor while its pmm-agent has disconnected, which `executor` would read
 * as "Ready" and this cell would not. The blocked reasons are the server's
 * own explanation, not re-derived here, so this cell can never disagree with
 * why the row actually says what it says.
 */
const AutomationCell = ({ row }: { row: OmHostRow }) => {
  if (row.automation_eligible) {
    return <Chip size="small" color="success" variant="outlined" label="Ready" />;
  }
  return (
    <Tooltip title={row.automation_blocked_reasons.join('; ') || 'Not eligible for automation.'}>
      <Chip size="small" color="warning" label="Needs attention" />
    </Tooltip>
  );
};

/** Whether this host can fetch packages, and what stopped it when it cannot. */
const RepoCell = ({ row }: { row: OmHostRow }) => {
  if (!row.repo) {
    return <Unavailable reason="probe_never_succeeded" />;
  }
  if (row.repo.reachable) {
    return (
      <Tooltip
        title={`${row.repo.url ?? 'The repository'} answered in ${row.repo.latency_ms ?? '?'} ms${
          row.repo.proxy ? ` via ${row.repo.proxy}` : ' with no proxy'
        }.`}
      >
        <Chip
          size="small"
          color="success"
          variant="outlined"
          label="Reachable"
        />
      </Tooltip>
    );
  }
  return (
    <Tooltip
      // The proxy is named whether or not one is set: a refused connection direct and
      // one through a broken proxy are the same message and different jobs.
      title={`${row.repo.error ?? 'The repository did not answer.'} ${
        row.repo.proxy ? `Proxy: ${row.repo.proxy}.` : 'No proxy configured.'
      }`}
    >
      <Chip size="small" color="error" variant="outlined" label="Unreachable" />
    </Tooltip>
  );
};

/**
 * Is there a database on this host, in the three answers that question has.
 *
 * The page's headline, and the reason it is not a boolean. PMM's own inventory cannot
 * tell a bare pmm-client host from an arbiter - same node type, same agents, no
 * services - so "no registered service" alone would report an arbiter as an empty
 * machine and invite someone to install over a port already in use.
 */
const DatabaseCell = ({ row }: { row: OmHostRow }) => {
  const extra =
    row.database_state === 'has_service'
      ? ` (${row.service_count})`
      : row.database_state === 'unregistered_only'
        ? ` (${row.unregistered_mongods.length})`
        : '';
  return (
    <Tooltip title={HOST_DATABASE_STATE_PHRASE[row.database_state]}>
      <Chip
        size="small"
        color={HOST_DATABASE_STATE_COLOR[row.database_state]}
        variant={row.database_state === 'has_service' ? 'filled' : 'outlined'}
        label={`${HOST_DATABASE_STATE_LABEL[row.database_state]}${extra}`}
      />
    </Tooltip>
  );
};

function useColumns(): MRT_ColumnDef<OmHostRow>[] {
  return useMemo(
    () => [
      { accessorKey: 'name', header: 'Host' },
      {
        accessorKey: 'address',
        header: 'Address',
        Cell: ({ row: { original } }) =>
          original.address ?? <Unavailable reason="not_applicable" />,
      },
      {
        id: 'database_state',
        accessorFn: (row) => HOST_DATABASE_STATE_LABEL[row.database_state],
        header: 'Database',
        Cell: ({ row: { original } }) => <DatabaseCell row={original} />,
      },
      {
        id: 'executor',
        // Sorted on the conclusion rather than on a boolean, so the unusable hosts
        // sort together whichever way they are unusable.
        accessorFn: (row) =>
          !row.executor.registered
            ? 'Not onboarded'
            : !row.executor.reachable
              ? 'Agent down'
              : !row.executor.driver_healthy
                ? 'Driver unhealthy'
                : 'Ready',
        header: 'Executor',
        Cell: ({ row: { original } }) => <ExecutorCell row={original} />,
      },
      {
        id: 'automation_eligible',
        accessorFn: (row) => (row.automation_eligible ? 'Ready' : 'Needs attention'),
        header: 'Automation',
        Cell: ({ row: { original } }) => <AutomationCell row={original} />,
      },
      {
        id: 'repo',
        accessorFn: (row) => row.repo?.latency_ms ?? Infinity,
        header: 'Repository',
        Cell: ({ row: { original } }) => <RepoCell row={original} />,
      },
      {
        accessorKey: 'os',
        header: 'OS',
        Cell: ({ row: { original } }) =>
          original.os ?? <Unavailable reason="probe_never_succeeded" />,
      },
      {
        accessorKey: 'kernel',
        header: 'Kernel',
        Cell: ({ row: { original } }) =>
          original.kernel ?? <Unavailable reason="probe_never_succeeded" />,
      },
      {
        id: 'collected',
        // Never-answered sorts last rather than first: as a timestamp string it would
        // sort beside the oldest row, which reads as "very stale" when it is "never".
        accessorFn: (row) =>
          ageSeconds(row.freshness.last_success_at) ?? Infinity,
        header: 'Collected',
        Cell: ({ row: { original } }) => {
          const age = ageSeconds(original.freshness.last_success_at);
          if (age == null) {
            return <Unavailable reason="probe_never_succeeded" />;
          }
          const since = ageSeconds(original.freshness.failing_since);
          return since == null ? (
            <>{formatCompactDuration(age)} ago</>
          ) : (
            <Tooltip
              title={`${original.freshness.last_error ?? 'The last probe failed.'} Failing for ${formatCompactDuration(
                since
              )}, ${original.freshness.consecutive_failures} attempts.`}
            >
              <Box
                component="span"
                sx={{ color: 'error.main', cursor: 'help' }}
              >
                {formatCompactDuration(age)} ago (failing)
              </Box>
            </Tooltip>
          );
        },
      },
      {
        accessorKey: 'node_id',
        header: 'Node ID',
      },
      {
        accessorKey: 'executor_host',
        header: 'Executor host',
        Cell: ({ row: { original } }) =>
          original.executor_host ?? <Unavailable reason="not_applicable" />,
      },
    ],
    []
  );
}

/**
 * What a host has on it, under its row.
 *
 * `GET /hosts` nests the services, so opening a row needs no second request - which is
 * why the API declined a `/hosts/{id}/services` collection. The unregistered mongods
 * are listed beside them because on a host with no registered service they are the
 * whole story.
 */
const HostDetail = ({ row }: { row: OmHostRow }) => {
  return (
    <Stack spacing={2} sx={{ p: 2 }}>
      <Box>
        <Typography variant="subtitle2" gutterBottom>
          Services PMM monitors ({row.services.length})
        </Typography>
        {row.services.length === 0 ? (
          <Typography variant="body2" color="text.secondary">
            None. PMM has no registered MongoDB service on this host.
          </Typography>
        ) : (
          <Stack spacing={0.5}>
            {row.services.map((service) => (
              <Typography variant="body2" key={service.service_id}>
                {service.name ?? service.service_id}
                {service.port ? `:${service.port}` : ''}
                {service.installed_version
                  ? ` — ${service.installed_version}`
                  : ''}
              </Typography>
            ))}
          </Stack>
        )}
      </Box>
      {row.unregistered_mongods.length > 0 && (
        <Box>
          <Typography variant="subtitle2" gutterBottom color="warning.main">
            Running but not registered ({row.unregistered_mongods.length})
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
            Found by the probe with no PMM service to match. An arbiter is the
            ordinary case: it holds no data, so PMM cannot authenticate against
            it to register one.
          </Typography>
          <Stack spacing={0.5}>
            {row.unregistered_mongods.map((mongod, index) => (
              <Typography variant="body2" key={`${mongod.port}-${index}`}>
                {mongod.program ?? 'mongod'}
                {mongod.port ? ` on :${mongod.port}` : ''}
                {mongod.config_path ? ` — ${mongod.config_path}` : ''}
              </Typography>
            ))}
          </Stack>
        </Box>
      )}
    </Stack>
  );
};

/**
 * The dialog that has to tell the truth about what deleting achieves, for one
 * host's row or several.
 *
 * Deleting is not suppression: an entity PMM still knows about comes straight back on
 * the next sweep. A confirm reading "delete this host?" invites the reader to believe
 * otherwise and use it as a mute exactly once, so this says what it is actually for -
 * clearing a row left behind when `pmm-agent setup --force` re-registered a node under
 * a new id, which leaves the old row with nothing to refresh it.
 *
 * The per-row Forget button and the bulk one share this dialog: the only real
 * difference is how many names are in the title and how many DELETE calls go out.
 * SEP has no batch-delete endpoint, so a bulk forget is N independent requests, not
 * one. They are dispatched together and awaited together; a partial failure keeps
 * the dialog open with the failures named, rather than closing over an incomplete
 * result the reader would have to notice was incomplete.
 *
 * `onClose` is "the reader backed out" -- Cancel and the dialog's own dismissal
 * (backdrop, Escape). `onForgotten` is "it worked" -- only called once every target
 * host is actually gone. They are two different callbacks because they mean two
 * different things to the caller: HostsPage clears the row-selection on the second,
 * never the first. Conflating them into one `onClose` was the bug -- backing out of
 * a destructive confirmation is not the same event as the destruction succeeding.
 */
const ForgetDialog = ({
  rows,
  onClose,
  onForgotten,
}: {
  rows: OmHostRow[];
  onClose: () => void;
  onForgotten: () => void;
}) => {
  const forget = useForgetHost();
  const [failures, setFailures] = useState<
    { nodeId: string; name: string; message: string }[]
  >([]);
  // Set while a batch is in flight. `forget.isPending` is one mutation's state, not
  // "any of N concurrent mutateAsync calls still pending" -- it can flip false as
  // soon as whichever call the shared observer last updated on settles, re-enabling
  // Forget while other DELETEs in the same batch are still out.
  const [busy, setBusy] = useState(false);
  // `rows.length === 0` below returns null rather than unmounting the component, so
  // `failures` would otherwise survive a close-and-reopen for an unrelated selection.
  // Left stale, it would filter `targets` (below) against node ids that do not exist
  // in the new `rows` at all -- an empty target list that "succeeds" without deleting
  // anything. `rows` is a fresh array reference exactly when the parent opens the
  // dialog for a new selection or closes it, never mid-retry, so resetting on it is
  // the right key.
  useEffect(() => {
    setFailures([]);
  }, [rows]);
  if (rows.length === 0) {
    return null;
  }
  // After a partial failure, retry dispatches only the rows named in `failures` --
  // the ones that already succeeded are gone from the estate, and re-sending their
  // DELETE would 404 and show up as a fresh, false failure for a host that is, in
  // fact, already forgotten.
  const failedIds = new Set(failures.map((failure) => failure.nodeId));
  const targets =
    failures.length > 0
      ? rows.filter((row) => failedIds.has(row.node_id))
      : rows;
  const totalServices = rows.reduce((sum, row) => sum + row.services.length, 0);
  const handleForget = async () => {
    setBusy(true);
    setFailures([]);
    try {
      const outcomes = await Promise.allSettled(
        targets.map((row) => forget.mutateAsync(row.node_id))
      );
      const failed = outcomes.flatMap((outcome, index) =>
        outcome.status === 'rejected'
          ? [
              {
                nodeId: targets[index].node_id,
                name: targets[index].name,
                message:
                  outcome.reason instanceof Error
                    ? outcome.reason.message
                    : String(outcome.reason),
              },
            ]
          : []
      );
      if (failed.length === 0) {
        onForgotten();
      } else {
        setFailures(failed);
      }
    } finally {
      setBusy(false);
    }
  };
  return (
    <Dialog open onClose={onClose} maxWidth="sm">
      <DialogTitle>
        {rows.length === 1
          ? `Forget ${rows[0].name}?`
          : `Forget ${rows.length} hosts?`}
      </DialogTitle>
      <DialogContent>
        <DialogContentText component="div">
          <p>
            This clears OM&apos;s row for{' '}
            {rows.length === 1 ? 'this host' : 'these hosts'} and the{' '}
            {totalServices} service row(s) on{' '}
            {rows.length === 1 ? 'it' : 'them'}, along with their probe history.
          </p>
          <p>
            <strong>
              It does not stop {rows.length === 1 ? 'this host' : 'these hosts'}{' '}
              being monitored.
            </strong>{' '}
            If PMM still has the node, the next sweep writes the row again. Use
            this to clear a duplicate left behind when a node was re-registered
            under a new ID.
          </p>
        </DialogContentText>
        {failures.map((failure) => (
          <Alert severity="error" key={failure.nodeId} sx={{ mt: 1 }}>
            {failure.name}: {failure.message}
          </Alert>
        ))}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button color="error" disabled={busy} onClick={handleForget}>
          Forget
        </Button>
      </DialogActions>
    </Dialog>
  );
};

const DEFAULT_MONGODB_VERSION = '7.0';

/**
 * Configure and trigger a single-host bootstrap.
 *
 * PMM-15347 PoC only: one host, one member, keyFile auth, TLS off. Two panes in
 * one dialog rather than a form that redirects on submit, because there is
 * exactly one thing to show once the mutation succeeds -- the generated admin
 * credentials -- and this is the last screen that will ever show them. Closing
 * the dialog (`onClose`) is available throughout; only a successful bootstrap
 * calls `onBootstrapped`, which is what clears the row selection the way
 * `ForgetDialog`'s `onForgotten` does.
 */
const BootstrapDialog = ({
  row,
  onClose,
  onBootstrapped,
}: {
  row: OmHostRow | null;
  onClose: () => void;
  onBootstrapped: () => void;
}) => {
  const bootstrap = useTriggerHostBootstrap();
  const [replicaSetName, setReplicaSetName] = useState('');
  const [mongodbVersion, setMongodbVersion] = useState(DEFAULT_MONGODB_VERSION);
  const [result, setResult] = useState<OmHostBootstrapAccepted | null>(null);

  // A fresh row (a new open, not a re-render of the same one) resets the form and
  // any previous result -- opening the dialog on a different host must not carry
  // over the last one's credentials or an in-flight mutation's error.
  useEffect(() => {
    setReplicaSetName('');
    setMongodbVersion(DEFAULT_MONGODB_VERSION);
    setResult(null);
    bootstrap.reset();
    // bootstrap is a fresh object every render (useMutation), so it is deliberately
    // left out of the dependency list -- including it would reset the form on every
    // keystroke-triggered re-render, not just on a genuinely new row.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [row]);

  if (!row) {
    return null;
  }

  const handleSubmit = async () => {
    const accepted = await bootstrap.mutateAsync({
      nodeId: row.node_id,
      replicaSetName,
      mongodbVersion,
    });
    setResult(accepted);
  };

  return (
    <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Bootstrap {row.name}</DialogTitle>
      <DialogContent>
        {result ? (
          <Stack spacing={2}>
            <Alert severity="success">
              Bootstrap queued as task {result.task_history_id}. Installing
              MongoDB and initializing the replica set takes a minute or two;
              the host&apos;s next probe will show the new service once it
              lands.
            </Alert>
            <Alert severity="warning">
              The admin password below is shown <strong>once</strong>. Nothing
              stores it after this dialog closes.
            </Alert>
            <TextField
              label="Admin username"
              value={result.admin_username}
              slotProps={{ input: { readOnly: true } }}
              fullWidth
            />
            <TextField
              label="Admin password"
              value={result.admin_password}
              slotProps={{ input: { readOnly: true, sx: { fontFamily: 'monospace' } } }}
              fullWidth
            />
          </Stack>
        ) : (
          <Stack spacing={2} sx={{ mt: 1 }}>
            <DialogContentText>
              Installs MongoDB on {row.name} through the Nomad client and
              initializes it as a single-member replica set. Proof-of-concept
              scope only -- keyFile auth, TLS off, no project or cluster yet.
            </DialogContentText>
            <TextField
              label="Replica set name"
              value={replicaSetName}
              onChange={(event) => setReplicaSetName(event.target.value)}
              required
              autoFocus
              fullWidth
            />
            <TextField
              label="MongoDB version"
              value={mongodbVersion}
              onChange={(event) => setMongodbVersion(event.target.value)}
              required
              fullWidth
              helperText="Only the major version selects the install source, e.g. 7.0."
            />
            {bootstrap.isError && (
              <Alert severity="error">{bootstrap.error.message}</Alert>
            )}
          </Stack>
        )}
      </DialogContent>
      <DialogActions>
        {result ? (
          <Button
            variant="contained"
            onClick={() => {
              onBootstrapped();
            }}
          >
            Done
          </Button>
        ) : (
          <>
            <Button onClick={onClose}>Cancel</Button>
            <Button
              variant="contained"
              disabled={
                bootstrap.isPending ||
                !replicaSetName.trim() ||
                !mongodbVersion.trim()
              }
              onClick={handleSubmit}
            >
              Bootstrap
            </Button>
          </>
        )}
      </DialogActions>
    </Dialog>
  );
};

/**
 * One row per host OM keeps, whether or not a database runs on it.
 *
 * The page §4 exists for. A host with no database is not an edge case here: it is
 * where a database can be installed, and it has no service to be discovered through,
 * which is why the estate keys hosts separately at all.
 */
export const HostsPage = () => {
  const { data, isPending, isError, error } = useOmInventoryHosts();
  const refresh = useRefreshInventory();
  // Any active run, matching InventoryPage's button: firing an estate-wide sweep into
  // one already in flight only earns a 409, and the row actions below cannot succeed
  // against a host that sweep already holds. The refetch when a sweep lands is the
  // estate query's own business now, so this page no longer arranges it.
  const refreshing = useIsEstateRefreshing();
  const [forgetting, setForgetting] = useState<OmHostRow[]>([]);
  const [bootstrapping, setBootstrapping] = useState<OmHostRow | null>(null);
  const [hostFilter, setHostFilter] = useState<HostFilter>('all');
  // Keyed by node_id (this table's getRowId), independent of which filter is
  // active — switching filters does not silently drop a selection made under a
  // different one.
  const [rowSelection, setRowSelection] = useState<Record<string, boolean>>({});
  const columns = useColumns();
  const rows = useMemo(() => toHostRows(data), [data]);
  // Filtered for the table only — the counts below stay whole-estate so switching
  // filters does not make the headline numbers look like they changed too.
  const filteredRows = useMemo(
    () => rows.filter((row) => matchesHostFilter(row, hostFilter)),
    [rows, hostFilter]
  );
  // Only the selected rows that are currently visible: a selection made under one
  // filter should not let a bulk action reach into rows the reader cannot see.
  const selectedRows = useMemo(
    () => filteredRows.filter((row) => rowSelection[row.node_id]),
    [filteredRows, rowSelection]
  );

  const counts = useMemo(
    () => ({
      total: rows.length,
      installable: rows.filter((row) => row.database_state === 'installable')
        .length,
      unregistered: rows.filter(
        (row) => row.database_state === 'unregistered_only'
      ).length,
      unusable: rows.filter(
        (row) =>
          !row.executor.registered ||
          !row.executor.reachable ||
          !row.executor.driver_healthy
      ).length,
      failing: rows.filter((row) => isFailing(row)).length,
      automationEligible: rows.filter((row) => row.automation_eligible).length,
    }),
    [rows]
  );

  const table = useMaterialReactTable({
    columns,
    data: filteredRows,
    // The server-issued id, not MRT's default row index. These rows are refetched on a
    // timer and again whenever a refresh lands, so an insertion or a reorder would move
    // an open detail panel onto a different host. The run and cluster tables already
    // key on their own ids for the same reason.
    getRowId: (row) => row.node_id,
    enablePagination: false,
    enableDensityToggle: false,
    enableExpanding: true,
    enableRowActions: true,
    enableRowSelection: true,
    positionActionsColumn: 'last',
    onRowSelectionChange: setRowSelection,
    state: { rowSelection },
    renderDetailPanel: ({ row }) => <HostDetail row={row.original} />,
    renderRowActions: ({ row }) => (
      <Stack direction="row" spacing={1}>
        {/* The span is load-bearing: MUI disables pointer events on a disabled
            ButtonBase, so a Tooltip wrapping the button directly never opens while
            a refresh is pending -- which is exactly when a reader wants to know why. */}
        <Tooltip title="Probe this host now. Dispatches a job and takes tens of seconds; the row updates when it lands.">
          <Box component="span">
            <Button
              size="small"
              disabled={refresh.isPending || refreshing}
              onClick={() => refresh.refreshHosts([row.original.node_id])}
            >
              Refresh
            </Button>
          </Box>
        </Tooltip>
        <Tooltip
          title={
            row.original.automation_eligible
              ? 'Install MongoDB on this host and initialize a single-member replica set (PoC).'
              : row.original.automation_blocked_reasons.join('; ')
          }
        >
          <Box component="span">
            <Button
              size="small"
              disabled={!row.original.automation_eligible}
              onClick={() => setBootstrapping(row.original)}
            >
              Bootstrap
            </Button>
          </Box>
        </Tooltip>
        <Button
          size="small"
          color="error"
          onClick={() => setForgetting([row.original])}
        >
          Forget
        </Button>
      </Stack>
    ),
    initialState: {
      density: 'compact',
      columnVisibility: HIDDEN_BY_DEFAULT,
      sorting: [{ id: 'name', desc: false }],
    },
  });

  if (isPending) {
    return <LinearProgress />;
  }

  if (isError) {
    return (
      <Alert severity="error">
        {/* An error here means SEP is unwell, and it renders inside the page rather
            than replacing it. That is the whole point of reaching the estate through
            pmm-managed: before the proxy, a sick SEP blanked the page entirely. */}
        {(error as Error)?.message ?? 'Could not load the host inventory.'}
      </Alert>
    );
  }

  return (
    <Box>
      <OmHeader
        title="Hosts"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every host OM knows about, including the ones with no database on
            them.
          </Typography>
        }
        actions={
          <Stack direction="row" alignItems="center" gap={2}>
            <HostFilterChips value={hostFilter} onChange={setHostFilter} />
            <Tooltip title="Probe every host. Dispatches one job per host and takes tens of seconds.">
              <Box component="span">
                <Button
                  variant="outlined"
                  disabled={refresh.isPending || refreshing}
                  onClick={() => refresh.refreshAll()}
                >
                  {refreshing ? 'Refreshing…' : 'Refresh all'}
                </Button>
              </Box>
            </Tooltip>
          </Stack>
        }
      />
      {refresh.isError && (
        <Alert
          severity={
            refresh.error instanceof OmApiError && refresh.error.status === 409
              ? 'info'
              : 'error'
          }
          sx={{ mb: 2 }}
        >
          {/* A 409 is an expected answer, not a fault: another sweep already holds
              these hosts, and the schedule starts one every few minutes. */}
          {refresh.error.message}
        </Alert>
      )}
      <Stack direction="row" spacing={3} sx={{ mb: 2, alignItems: 'center' }}>
        <Typography variant="body2">
          <strong>{counts.total}</strong> hosts
        </Typography>
        <Tooltip title="PMM-Client connected, and the Nomad executor reachable and driver-healthy.">
          <Typography variant="body2" sx={{ cursor: 'help' }}>
            <strong>{counts.automationEligible}</strong> eligible for automation
          </Typography>
        </Tooltip>
        <Tooltip title="No registered service and no mongod found - a host a database could be installed on.">
          <Typography variant="body2" sx={{ cursor: 'help' }}>
            <strong>{counts.installable}</strong> with no database
          </Typography>
        </Tooltip>
        {counts.unregistered > 0 && (
          <Tooltip title="A mongod is running that PMM has no service for. Not an empty host.">
            <Typography
              variant="body2"
              color="warning.main"
              sx={{ cursor: 'help' }}
            >
              <strong>{counts.unregistered}</strong> unregistered
            </Typography>
          </Tooltip>
        )}
        {counts.unusable > 0 && (
          <Typography variant="body2" color="error.main">
            <strong>{counts.unusable}</strong> cannot be probed
          </Typography>
        )}
        {counts.failing > 0 && (
          <Typography variant="body2" color="error.main">
            <strong>{counts.failing}</strong> failing
          </Typography>
        )}
      </Stack>
      {selectedRows.length > 0 && (
        <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2 }}>
          <Typography variant="body2">
            <strong>{selectedRows.length}</strong> selected
          </Typography>
          <Tooltip title="Probe every selected host. Dispatches one job per host and takes tens of seconds.">
            <Box component="span">
              <Button
                size="small"
                variant="outlined"
                disabled={refresh.isPending || refreshing}
                onClick={() => {
                  refresh.refreshHosts(selectedRows.map((row) => row.node_id));
                  setRowSelection({});
                }}
              >
                Refresh selected
              </Button>
            </Box>
          </Tooltip>
          <Button
            size="small"
            variant="outlined"
            color="error"
            onClick={() => setForgetting(selectedRows)}
          >
            Forget selected
          </Button>
        </Stack>
      )}
      <MaterialReactTable table={table} />
      <ForgetDialog
        rows={forgetting}
        onClose={() => setForgetting([])}
        onForgotten={() => {
          setForgetting([]);
          setRowSelection({});
        }}
      />
      <BootstrapDialog
        row={bootstrapping}
        onClose={() => setBootstrapping(null)}
        onBootstrapped={() => setBootstrapping(null)}
      />
    </Box>
  );
};
