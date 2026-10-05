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
import { EmptyState } from './components/EmptyState';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import {
  MaterialReactTable,
  useMaterialReactTable,
  type MRT_ColumnDef,
} from 'material-react-table';
import { PROCESS_ROLE_LABEL } from './constants';
import { NESTED_TABLE_WRAPPER, STICKY_IDENTITY_CELL } from './nestedTable';
import { SnapshotBar } from './components/SnapshotBar';
import { ClusterHealthBadge, StatusBadge } from './components/HealthBadge';
import { MemberState } from './components/MemberState';
import { ServiceLink } from './components/ServiceLink';
import { Duration, Percent } from './components/Metric';
import { Unavailable } from './components/Unavailable';
import { useOmTopology } from './topologyHooks';
import { pluralize } from './format';
import {
  clusterHealthRank,
  downFirst,
  toEnvironmentSections,
} from './topology';
import type {
  OmClusterHealth,
  OmClusterRow,
  OmEnvironmentSection,
  OmProcessRole,
} from './types';

/** Label for an environment or cluster the services carry no name for. */
const UNNAMED_ENVIRONMENT = 'Unassigned environment';

/** `3 mongod · 1 Router`, in the document's own role order. */
function describeRoles(roles: Partial<Record<OmProcessRole, number>>): string {
  return Object.entries(roles)
    .map(
      ([role, count]) =>
        `${count} ${PROCESS_ROLE_LABEL[role as OmProcessRole] ?? role}`
    )
    .join(' · ');
}

/**
 * A count that stays legible when it is zero.
 *
 * Zero down is the good news and should read as such; zero up in a cluster that has
 * services is the whole point of the page, so it keeps the error colour.
 */
const Count = ({ value, tone }: { value: number; tone: 'up' | 'down' }) => {
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
};

/** Columns for one environment's cluster table. */
function useColumns(): MRT_ColumnDef<OmClusterRow>[] {
  return useMemo(
    () => [
      {
        accessorKey: 'cluster_name',
        header: 'Cluster',
        Cell: ({ row: { original } }) =>
          original.cluster_name ?? <Unavailable reason="not_applicable" />,
      },
      {
        accessorKey: 'health',
        header: 'Health',
        // Worst first ascending, so one click brings trouble to the top.
        sortingFn: (a, b, columnId) =>
          clusterHealthRank(a.getValue<OmClusterHealth>(columnId)) -
          clusterHealthRank(b.getValue<OmClusterHealth>(columnId)),
        Cell: ({ row: { original } }) => (
          <ClusterHealthBadge health={original.health} />
        ),
      },
      { accessorKey: 'total_services', header: 'Services' },
      {
        accessorKey: 'up_services',
        header: 'Up',
        Cell: ({ row: { original } }) => (
          <Count value={original.up_services} tone="up" />
        ),
      },
      {
        accessorKey: 'down_services',
        header: 'Down',
        Cell: ({ row: { original } }) => (
          <Count value={original.down_services} tone="down" />
        ),
      },
      {
        accessorFn: (row) => describeRoles(row.by_process_role),
        id: 'roles',
        header: 'Process',
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
const ClusterServices = ({ cluster }: { cluster: OmClusterRow }) => {
  if (!cluster.services.length) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
        This cluster has no services in the current snapshot.
      </Typography>
    );
  }
  return (
    <Box sx={NESTED_TABLE_WRAPPER}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell sx={STICKY_IDENTITY_CELL}>Service</TableCell>
            <TableCell>Host</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Member state</TableCell>
            <TableCell>Process</TableCell>
            <TableCell>Version</TableCell>
            <TableCell>CPU</TableCell>
            <TableCell>Conn. free</TableCell>
            <TableCell>Repl. lag</TableCell>
            <TableCell>Oplog window</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {downFirst(cluster.services).map((service) => (
            <TableRow key={service.service_name}>
              <TableCell sx={STICKY_IDENTITY_CELL}>
                <ServiceLink serviceName={service.service_name} />
              </TableCell>
              <TableCell>
                {service.host ?? <Unavailable reason="service_not_observed" />}
              </TableCell>
              <TableCell>
                <StatusBadge status={service.status} />
              </TableCell>
              <TableCell>
                <MemberState service={service} />
              </TableCell>
              <TableCell>
                {PROCESS_ROLE_LABEL[service.process_role] ??
                  service.process_role}
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
};

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
const EnvironmentTable = ({ section }: { section: OmEnvironmentSection }) => {
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
    // The opaque server-issued id, not the label: two clusters can share a label (a
    // sandbox's second generation reusing a name is not hypothetical here) or carry
    // none at all, and MRT uses this for expansion and selection state across
    // refetches -- an index-based fallback would misattribute that state the moment
    // the row order shifted between polls.
    getRowId: (row) => row.id,
    renderDetailPanel: ({ row }) => <ClusterServices cluster={row.original} />,
    initialState: {
      density: 'compact',
      // Trouble first: a degraded or down cluster leads its environment.
      sorting: [
        { id: 'health', desc: false },
        { id: 'cluster_name', desc: false },
      ],
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
            <strong>{section.clusters.length}</strong>{' '}
            {pluralize(section.clusters.length, 'cluster')}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            <strong>{section.total_services}</strong>{' '}
            {pluralize(section.total_services, 'service')}
          </Typography>
          <Typography
            variant="body2"
            color={section.up_services ? 'success.main' : 'text.secondary'}
          >
            <strong>{section.up_services}</strong> up
          </Typography>
          <Typography
            variant="body2"
            color={section.down_services ? 'error.main' : 'text.secondary'}
          >
            <strong>{section.down_services}</strong> down
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
};

/** Fleet-level counts, so the headline numbers need no reading of rows. */
const Counts = ({
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
}) => {
  return (
    <Stack direction="row" spacing={3} flexWrap="wrap">
      <Typography variant="body2">
        <strong>{environments}</strong> {pluralize(environments, 'environment')}
      </Typography>
      <Typography variant="body2">
        <strong>{clusters}</strong> {pluralize(clusters, 'cluster')}
      </Typography>
      <Typography variant="body2">
        <strong>{total}</strong> {pluralize(total, 'service')}
      </Typography>
      <Typography
        variant="body2"
        color={up ? 'success.main' : 'text.secondary'}
      >
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
};

/**
 * The estate one level above the service table: a table per environment, a row per
 * cluster, and each row unfolds into the services it was rolled up from.
 *
 * The document is a nested `environments -> clusters -> services` tree and this is the
 * reading that keeps the nesting whole. Topology renders the same snapshot flat, one
 * row per service and every field it carries, for anyone who needs to sort or filter
 * across the estate rather than read it by environment.
 */
export const FleetClustersTab = () => {
  const { data, isPending, isError, error } = useOmTopology();
  const sections = useMemo(() => toEnvironmentSections(data), [data]);

  if (isPending) {
    return <LinearProgress />;
  }

  if (isError) {
    // Just the alert: the header, and the Sync action that is the way out of the
    // expected first-run 503, belong to FleetPage and render above whichever tab is
    // open. This tab is the index route, so a fresh install lands here first.
    return (
      <Alert severity="error">
        {(error as Error)?.message ?? 'Could not load the fleet.'}
      </Alert>
    );
  }

  return (
    <Stack gap={3}>
      <Stack gap={1}>
        <SnapshotBar envelope={data.snapshot} />
        <Counts
          environments={data.summary.environments}
          clusters={data.summary.clusters}
          total={data.summary.total_services}
          up={data.summary.up_services}
          down={data.summary.down_services}
        />
      </Stack>

      {sections.length === 0 ? (
        <EmptyState title="No MongoDB clusters yet">
          This page shows every MongoDB cluster PMM monitors, and the health of
          each one&apos;s members. It is empty because PMM has no MongoDB
          services registered yet - add one, and it appears here on the next
          refresh.
        </EmptyState>
      ) : (
        // Indexed fallback: two sibling sections with no env_name would otherwise
        // share a React key. Environments carry no server-issued id the way clusters
        // now do -- see EnvironmentTable's getRowId -- so this stays index-based.
        sections.map((section, index) => (
          <EnvironmentTable
            key={section.env_name ?? `unnamed-${index}`}
            section={section}
          />
        ))
      )}
    </Stack>
  );
};
