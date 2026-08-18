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

import { useMemo, useState } from 'react';
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
import { PomHeader } from './components/PomHeader';
import { Unavailable } from './components/Unavailable';
import { formatDuration } from './format';
import { ageSeconds, isFailing, toHostRows } from './inventory';
import {
  useForgetHost,
  usePomInventoryHosts,
  useRefreshInventory,
} from './inventoryHooks';
import { PomApiError } from './hooks';
import type { PomHostRow } from './types';

/** Identifiers and long text the table carries but does not open with. */
const HIDDEN_BY_DEFAULT = {
  node_id: false,
  address: false,
  kernel: false,
};

/**
 * Why nothing can run on a host, in the three ways it can be true.
 *
 * Kept as one cell rather than three columns because they are not independent: a host
 * that is not registered cannot be reachable, and reading three booleans to reach one
 * conclusion is work the page should do for its reader. The distinction that matters
 * is what to go and do, so that is what the cell says.
 */
function ExecutorCell({ row }: { row: PomHostRow }) {
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
}

/** Whether this host can fetch packages, and what stopped it when it cannot. */
function RepoCell({ row }: { row: PomHostRow }) {
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
        <Box component="span" sx={{ cursor: 'help' }}>
          {row.repo.latency_ms == null
            ? 'Reachable'
            : `${row.repo.latency_ms} ms`}
        </Box>
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
}

/**
 * Is there a database on this host, in the three answers that question has.
 *
 * The page's headline, and the reason it is not a boolean. PMM's own inventory cannot
 * tell a bare pmm-client host from an arbiter - same node type, same agents, no
 * services - so "no registered service" alone would report an arbiter as an empty
 * machine and invite someone to install over a port already in use.
 */
function DatabaseCell({ row }: { row: PomHostRow }) {
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
}

function useColumns(): MRT_ColumnDef<PomHostRow>[] {
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
            <>{formatDuration(age)} ago</>
          ) : (
            <Tooltip
              title={`${original.freshness.last_error ?? 'The last probe failed.'} Failing for ${formatDuration(
                since
              )}, ${original.freshness.consecutive_failures} attempts.`}
            >
              <Box
                component="span"
                sx={{ color: 'error.main', cursor: 'help' }}
              >
                {formatDuration(age)} ago (failing)
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
function HostDetail({ row }: { row: PomHostRow }) {
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
}

/**
 * The dialog that has to tell the truth about what deleting achieves.
 *
 * Deleting is not suppression: an entity PMM still knows about comes straight back on
 * the next sweep. A confirm reading "delete this host?" invites the reader to believe
 * otherwise and use it as a mute exactly once, so this says what it is actually for -
 * clearing a row left behind when `pmm-agent setup --force` re-registered a node under
 * a new id, which leaves the old row with nothing to refresh it.
 */
function ForgetDialog({
  row,
  onClose,
}: {
  row: PomHostRow | null;
  onClose: () => void;
}) {
  const forget = useForgetHost();
  if (!row) {
    return null;
  }
  return (
    <Dialog open onClose={onClose} maxWidth="sm">
      <DialogTitle>Forget {row.name}?</DialogTitle>
      <DialogContent>
        <DialogContentText component="div">
          <p>
            This clears POM&apos;s row for this host and the{' '}
            {row.services.length} service row(s) on it, along with their probe
            history.
          </p>
          <p>
            <strong>It does not stop this host being monitored.</strong> If PMM
            still has the node, the next sweep writes the row again. Use this to
            clear a duplicate left behind when a node was re-registered under a
            new ID.
          </p>
        </DialogContentText>
        {forget.isError && (
          <Alert severity="error">{forget.error.message}</Alert>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          color="error"
          disabled={forget.isPending}
          onClick={() =>
            forget.mutate(row.node_id, { onSuccess: () => onClose() })
          }
        >
          Forget
        </Button>
      </DialogActions>
    </Dialog>
  );
}

/**
 * One row per host POM keeps, whether or not a database runs on it.
 *
 * The page §4 exists for. A host with no database is not an edge case here: it is
 * where a database can be installed, and it has no service to be discovered through,
 * which is why the estate keys hosts separately at all.
 */
export function HostsPage() {
  const { data, isPending, isError, error } = usePomInventoryHosts();
  const refresh = useRefreshInventory();
  const [forgetting, setForgetting] = useState<PomHostRow | null>(null);
  const columns = useColumns();
  const rows = useMemo(() => toHostRows(data), [data]);

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
    }),
    [rows]
  );

  const table = useMaterialReactTable({
    columns,
    data: rows,
    enablePagination: false,
    enableDensityToggle: false,
    enableExpanding: true,
    enableRowActions: true,
    positionActionsColumn: 'last',
    renderDetailPanel: ({ row }) => <HostDetail row={row.original} />,
    renderRowActions: ({ row }) => (
      <Stack direction="row" spacing={1}>
        <Tooltip title="Probe this host now. Dispatches a job and takes tens of seconds; the row updates when it lands.">
          <Button
            size="small"
            disabled={refresh.isPending}
            onClick={() => refresh.mutate([row.original.node_id])}
          >
            Refresh
          </Button>
        </Tooltip>
        <Button
          size="small"
          color="error"
          onClick={() => setForgetting(row.original)}
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
      <PomHeader
        title="Hosts"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every host POM knows about, including the ones with no database on
            them.
          </Typography>
        }
        actions={
          <Tooltip title="Probe every host. Dispatches one job per host and takes tens of seconds.">
            <Button
              variant="outlined"
              disabled={refresh.isPending}
              onClick={() => refresh.mutate(undefined)}
            >
              Refresh all
            </Button>
          </Tooltip>
        }
      />
      {refresh.isError && (
        <Alert
          severity={
            refresh.error instanceof PomApiError && refresh.error.status === 409
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
      <MaterialReactTable table={table} />
      <ForgetDialog row={forgetting} onClose={() => setForgetting(null)} />
    </Box>
  );
}
