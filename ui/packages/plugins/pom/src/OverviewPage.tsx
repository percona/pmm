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

import { useId, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  ButtonBase,
  Collapse,
  LinearProgress,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import {
  MaterialReactTable,
  useMaterialReactTable,
  type MRT_ColumnDef,
} from 'material-react-table';
import { PROCESS_ROLE_LABEL } from './constants';
import { PomHeader } from './components/PomHeader';
import { SnapshotBar } from './components/SnapshotBar';
import { StatusBadge } from './components/HealthBadge';
import { SyncButton } from './components/SyncButton';
import { Duration, Percent } from './components/Metric';
import { Unavailable } from './components/Unavailable';
import { toEnvironmentSections, usePomTopology } from './hooks';
import type {
  PomClusterRow,
  PomEnvironmentSection,
  PomProcessRole,
} from './types';

/** Label for an environment or cluster the services carry no name for. */
const UNNAMED_ENVIRONMENT = 'No environment';

/** `3 mongod · 1 Router`, in the document's own role order. */
function describeRoles(roles: Partial<Record<PomProcessRole, number>>): string {
  return Object.entries(roles)
    .map(
      ([role, count]) =>
        `${count} ${PROCESS_ROLE_LABEL[role as PomProcessRole] ?? role}`
    )
    .join(' · ');
}

/** `1 PRIMARY · 2 SECONDARY`, worst-known ordering left to the reader. */
function describeStates(states: Record<string, number>): string {
  return Object.entries(states)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([state, count]) => `${count} ${state}`)
    .join(' · ');
}

/**
 * A count that stays legible when it is zero.
 *
 * Zero down is the good news and should read as such; zero up in a cluster that has
 * services is the whole point of the page, so it keeps the error colour.
 */
function Count({ value, tone }: { value: number; tone: 'up' | 'down' }) {
  const colour =
    tone === 'up'
      ? value > 0
        ? 'success.main'
        : 'error.main'
      : value > 0
        ? 'error.main'
        : 'text.secondary';
  return (
    <Typography variant="body2" component="span" color={colour}>
      {value}
    </Typography>
  );
}

/** Columns for one environment's cluster table. */
function useColumns(): MRT_ColumnDef<PomClusterRow>[] {
  return useMemo(
    () => [
      {
        accessorKey: 'cluster_name',
        header: 'Cluster',
        Cell: ({ row: { original } }) =>
          original.cluster_name ?? <Unavailable reason="not_applicable" />,
      },
      { accessorKey: 'services_total', header: 'Services' },
      {
        accessorKey: 'services_up',
        header: 'Up',
        Cell: ({ row: { original } }) => (
          <Count value={original.services_up} tone="up" />
        ),
      },
      {
        accessorKey: 'services_down',
        header: 'Down',
        Cell: ({ row: { original } }) => (
          <Count value={original.services_down} tone="down" />
        ),
      },
      {
        accessorFn: (row) => describeRoles(row.by_process_role),
        id: 'roles',
        header: 'Roles',
      },
      {
        accessorFn: (row) => describeStates(row.by_state),
        id: 'states',
        header: 'Member states',
        Cell: ({ row: { original } }) =>
          // A cluster of routers or a standalone has no replica-set state at all,
          // which is a different statement from "we could not see it".
          describeStates(original.by_state) || (
            <Unavailable reason="not_applicable" />
          ),
      },
      {
        accessorFn: (row) => row.versions.join(', '),
        id: 'versions',
        header: 'Versions',
        Cell: ({ row: { original } }) => {
          if (!original.versions.length) {
            return <Unavailable reason="service_not_observed" />;
          }
          if (original.versions.length === 1) {
            return original.versions[0];
          }
          // More than one running version in one cluster is worth reading as a
          // finding, not as a longer cell.
          return (
            <Tooltip title={original.versions.join(', ')}>
              <Typography variant="body2" component="span" color="warning.main">
                Mixed ({original.versions.length})
              </Typography>
            </Tooltip>
          );
        },
      },
      {
        accessorKey: 'max_replication_lag_seconds',
        header: 'Max repl. lag',
        Cell: ({ row: { original } }) => (
          <Duration value={original.max_replication_lag_seconds} />
        ),
      },
      {
        accessorKey: 'min_oplog_window_seconds',
        header: 'Min oplog window',
        Cell: ({ row: { original } }) => (
          <Duration value={original.min_oplog_window_seconds} />
        ),
      },
    ],
    []
  );
}

/**
 * The services of one cluster, shown when its row is unfolded.
 *
 * Deliberately a plain table rather than a nested data grid: this is the roll-up
 * being shown its working, so it needs no second set of sorters and filters. The
 * fields the snapshot carries beyond these live on Topology.
 */
function ClusterServices({ cluster }: { cluster: PomClusterRow }) {
  if (!cluster.services.length) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
        This cluster has no services in the current snapshot.
      </Typography>
    );
  }
  return (
    <Box sx={{ p: 2, overflowX: 'auto' }}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Service</TableCell>
            <TableCell>Host</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Role</TableCell>
            <TableCell>Member state</TableCell>
            <TableCell>Version</TableCell>
            <TableCell>CPU</TableCell>
            <TableCell>Conn. free</TableCell>
            <TableCell>Repl. lag</TableCell>
            <TableCell>Oplog window</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {cluster.services.map((service) => (
            <TableRow key={service.service_name}>
              <TableCell>{service.service_name}</TableCell>
              <TableCell>
                {service.host ?? <Unavailable reason="service_not_observed" />}
              </TableCell>
              <TableCell>
                <StatusBadge status={service.status} />
              </TableCell>
              <TableCell>
                {PROCESS_ROLE_LABEL[service.process_role] ??
                  service.process_role}
              </TableCell>
              <TableCell>
                {service.state ?? <Unavailable reason="not_applicable" />}
              </TableCell>
              <TableCell>
                {service.version ?? (
                  <Unavailable reason="service_not_observed" />
                )}
              </TableCell>
              <TableCell>
                <Percent value={service.cpu_usage_percent} />
              </TableCell>
              <TableCell>
                <Percent value={service.connections_free_percent} />
              </TableCell>
              <TableCell>
                <Duration value={service.replication_lag_seconds} />
              </TableCell>
              <TableCell>
                <Duration value={service.oplog_window_seconds} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  );
}

/**
 * One environment, as its own foldable table.
 *
 * A component per environment rather than a loop over table instances: each table
 * owns its sorting, filtering, expanded rows and fold state, and hooks cannot be
 * called in a loop anyway.
 *
 * Folded away, an environment still shows its counts — the header is what makes
 * folding useful on an estate with more environments than fit on a screen, and a
 * header that hid its numbers would just be a heading.
 */
function EnvironmentTable({ section }: { section: PomEnvironmentSection }) {
  const [open, setOpen] = useState(true);
  const regionId = useId();
  const columns = useColumns();
  const table = useMaterialReactTable({
    columns,
    data: section.clusters,
    enablePagination: false,
    enableDensityToggle: false,
    enableExpanding: true,
    enableTopToolbar: false,
    getRowId: (row) => row.cluster_name ?? UNNAMED_ENVIRONMENT,
    renderDetailPanel: ({ row }) => <ClusterServices cluster={row.original} />,
    initialState: {
      density: 'compact',
      sorting: [{ id: 'cluster_name', desc: false }],
    },
  });

  return (
    <Stack gap={1}>
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        gap={2}
        flexWrap="wrap"
      >
        <ButtonBase
          onClick={() => setOpen(!open)}
          aria-expanded={open}
          aria-controls={regionId}
          sx={{ gap: 1, px: 0.5, borderRadius: 1 }}
        >
          <ExpandMoreIcon
            fontSize="small"
            sx={{
              transition: 'transform 150ms',
              transform: open ? 'none' : 'rotate(-90deg)',
            }}
          />
          <Typography variant="h5" component="h2">
            {section.env_name ?? UNNAMED_ENVIRONMENT}
          </Typography>
        </ButtonBase>
        <Stack direction="row" spacing={2} flexWrap="wrap">
          <Typography variant="body2" color="text.secondary">
            <strong>{section.clusters.length}</strong> clusters
          </Typography>
          <Typography variant="body2" color="text.secondary">
            <strong>{section.services_total}</strong> services
          </Typography>
          <Typography variant="body2" color="success.main">
            <strong>{section.services_up}</strong> up
          </Typography>
          <Typography
            variant="body2"
            color={section.services_down ? 'error.main' : 'text.secondary'}
          >
            <strong>{section.services_down}</strong> down
          </Typography>
        </Stack>
      </Stack>
      {/* `unmountOnExit` keeps a folded estate off the DOM; the table's own state
          lives in the hook instance above, so sorting and unfolded clusters survive
          the round trip. */}
      <Collapse in={open} id={regionId} unmountOnExit>
        <MaterialReactTable table={table} />
      </Collapse>
    </Stack>
  );
}

/** Fleet-level counts, so the headline numbers need no reading of rows. */
function Counts({
  environments,
  clusters,
  total,
  up,
  down,
}: {
  environments: number;
  clusters: number;
  total: number;
  up: number;
  down: number;
}) {
  return (
    <Stack direction="row" spacing={3} flexWrap="wrap">
      <Typography variant="body2">
        <strong>{environments}</strong> environments
      </Typography>
      <Typography variant="body2">
        <strong>{clusters}</strong> clusters
      </Typography>
      <Typography variant="body2">
        <strong>{total}</strong> services
      </Typography>
      <Typography variant="body2" color="success.main">
        <strong>{up}</strong> up
      </Typography>
      <Typography
        variant="body2"
        color={down ? 'error.main' : 'text.secondary'}
      >
        <strong>{down}</strong> down
      </Typography>
    </Stack>
  );
}

/**
 * The estate one level above the service table: a table per environment, a row per
 * cluster, and each row unfolds into the services it was rolled up from.
 *
 * The document is a nested `environments -> clusters -> services` tree and this is the
 * reading that keeps the nesting whole. Topology renders the same snapshot flat, one
 * row per service and every field it carries, for anyone who needs to sort or filter
 * across the estate rather than read it by environment.
 */
export function OverviewPage() {
  const { data, isPending, isError, error } = usePomTopology();
  const sections = useMemo(() => toEnvironmentSections(data), [data]);

  if (isPending) {
    return <LinearProgress />;
  }

  if (isError) {
    return (
      <Alert severity="error">
        {/* A 503 here is the expected first-run state, not a fault: the API says so
            when no discovery has completed, and Sync is the way out. */}
        {(error as Error)?.message ?? 'Could not load the topology.'}
      </Alert>
    );
  }

  return (
    <Stack gap={3}>
      <Stack gap={1}>
        <PomHeader
          title="PSMDB OpenManager"
          subtitle={
            <Typography variant="body2" color="text.secondary">
              Every monitored MongoDB cluster, one table per environment. Unfold
              a cluster to see its services.
            </Typography>
          }
          actions={<SyncButton />}
        />
        <SnapshotBar envelope={data.snapshot} />
        <Counts
          environments={data.summary.environments}
          clusters={data.summary.clusters}
          total={data.summary.services_total}
          up={data.summary.services_up}
          down={data.summary.services_down}
        />
      </Stack>

      {sections.length === 0 ? (
        <Alert severity="info">
          The snapshot has no environments. Sync to rebuild it.
        </Alert>
      ) : (
        sections.map((section) => (
          <EnvironmentTable
            key={section.env_name ?? UNNAMED_ENVIRONMENT}
            section={section}
          />
        ))
      )}
    </Stack>
  );
}
