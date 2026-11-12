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

import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import type { MRT_ColumnDef } from 'material-react-table';
import { Age } from '../../../components/Age';
import { ClusterHealthBadge } from '../../../components/HealthBadge';
import { Duration, Percent } from '../../../components/Metric';
import { Unavailable } from '../../../components/Unavailable';
import { clusterHealthRank } from '../../../topology';
import type { OmClusterHealth, OmClusterRow } from '../../../types';
import {
  TYPE_FILTER_VALUE,
  UNNAMED_ENVIRONMENT_VALUE,
} from '../../FleetClustersTab.constants';
import { Messages } from '../../FleetClustersTab.messages';
import {
  GROUP_VALUE,
  anyOf,
  compareVersionLists,
  describeRoles,
  environmentLabel,
  typeLabel,
} from '../../FleetClustersTab.utils';
import { ClusterVersionCell } from '../cluster-version-cell';

const { columns, columnTooltips } = Messages;

export const CLUSTER_COLUMNS: MRT_ColumnDef<OmClusterRow>[] = [
  {
    accessorFn: GROUP_VALUE.env_name,
    id: 'env_name',
    header: columns.environment,
    size: 160,
    filterFn: anyOf((row) => row.env_name || UNNAMED_ENVIRONMENT_VALUE),
    Cell: ({ row: { original } }) => environmentLabel(original),
  },
  {
    accessorKey: 'cluster_name',
    header: columns.cluster,
    size: 200,
    // The only column the search reads.
    enableGlobalFilter: true,
    Cell: ({ row: { original } }) => (
      <Stack
        direction="row"
        alignItems="baseline"
        columnGap={1}
        flexWrap="wrap"
      >
        {original.cluster_name ?? <Unavailable reason="not_applicable" />}
        {typeLabel(original) && (
          <Typography
            variant="caption"
            color="text.secondary"
            data-testid="om-cluster-type"
          >
            {typeLabel(original)}
          </Typography>
        )}
      </Stack>
    ),
  },
  {
    accessorFn: (row) => typeLabel(row),
    id: 'type',
    header: columns.type,
    size: 120,
    sortUndefined: 'last',
    filterFn: anyOf((row) => TYPE_FILTER_VALUE[row.type]),
    Cell: ({ row: { original } }) =>
      typeLabel(original) ?? <Unavailable reason="not_applicable" />,
  },
  {
    accessorKey: 'health',
    header: columns.health,
    size: 130,
    filterFn: anyOf((row) => row.health),
    // Worst first ascending
    sortingFn: (a, b, columnId) =>
      clusterHealthRank(a.getValue<OmClusterHealth>(columnId)) -
      clusterHealthRank(b.getValue<OmClusterHealth>(columnId)),
    Cell: ({ row: { original } }) => (
      <ClusterHealthBadge health={original.health} />
    ),
  },
  {
    accessorFn: (row) => row.up_services,
    id: 'members',
    header: columns.members,
    size: 120,
    Header: () => (
      <Tooltip title={columnTooltips.members}>
        <span>{columns.members}</span>
      </Tooltip>
    ),
    Cell: ({ row: { original } }) =>
      `${original.up_services}/${original.total_services}`,
  },
  { accessorKey: 'total_services', header: columns.services, size: 100 },
  {
    accessorFn: (row) => describeRoles(row.by_process_role),
    id: 'roles',
    header: columns.process,
    size: 130,
  },
  {
    accessorFn: (row) => row.versions.join(', ') || undefined,
    id: 'versions',
    header: columns.versions,
    size: 130,
    sortUndefined: 'last',
    // As versions, not strings: 7.0.9-1 is older than 7.0.24-13.
    sortingFn: (a, b) =>
      compareVersionLists(a.original.versions, b.original.versions),
    Cell: ({ row: { original } }) => <ClusterVersionCell cluster={original} />,
  },
  {
    accessorKey: 'max_replication_lag_seconds',
    header: columns.maxReplicationLag,
    size: 130,
    Cell: ({ row: { original } }) => (
      <Duration value={original.max_replication_lag_seconds} />
    ),
  },
  {
    accessorKey: 'min_oplog_window_seconds',
    header: columns.minOplogWindow,
    size: 150,
    Cell: ({ row: { original } }) => (
      <Duration value={original.min_oplog_window_seconds} />
    ),
  },
  {
    accessorFn: (row) =>
      row.last_seen_at ? Date.parse(row.last_seen_at) : undefined,
    id: 'last_seen_at',
    header: columns.lastSeen,
    size: 120,
    sortUndefined: 'last',
    Header: () => (
      <Tooltip title={columnTooltips.lastSeen}>
        <span>{columns.lastSeen}</span>
      </Tooltip>
    ),
    Cell: ({ row: { original } }) =>
      original.last_seen_at ? (
        <Age value={original.last_seen_at} />
      ) : (
        <Unavailable reason="service_not_observed" />
      ),
  },
  {
    accessorFn: (row) => row.max_cpu_percent ?? undefined,
    id: 'max_cpu_percent',
    header: columns.maxCpu,
    size: 110,
    sortUndefined: 'last',
    Cell: ({ row: { original } }) =>
      original.max_cpu_percent == null ? (
        <Unavailable reason="metric_not_collected" />
      ) : (
        <Percent value={original.max_cpu_percent} />
      ),
  },
  {
    accessorFn: (row) => row.min_connections_free_percent ?? undefined,
    id: 'min_connections_free_percent',
    header: columns.minConnectionsFree,
    size: 140,
    sortUndefined: 'last',
    Header: () => (
      <Tooltip title={columnTooltips.minConnectionsFree}>
        <span>{columns.minConnectionsFree}</span>
      </Tooltip>
    ),
    Cell: ({ row: { original } }) =>
      original.min_connections_free_percent == null ? (
        <Unavailable reason="metric_not_collected" />
      ) : (
        <Percent value={original.min_connections_free_percent} />
      ),
  },
];
