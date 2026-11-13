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

import type { ReactNode } from 'react';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import type { MRT_ColumnDef } from 'material-react-table';
import { Age } from '../../../components/Age';
import { Percent } from '../../../components/Metric';
import { Unavailable } from '../../../components/Unavailable';
import { formatCompactDuration } from '../../../format';
import { clusterHealthRank } from '../../../topology';
import type { OmClusterHealth, OmClusterRow } from '../../../types';
import {
  LAG_ALERT_SECONDS,
  OPLOG_WARN_SECONDS,
  TYPE_FILTER_VALUE,
  UNNAMED_ENVIRONMENT_VALUE,
} from '../../FleetClustersTab.constants';
import { Messages } from '../../FleetClustersTab.messages';
import {
  GROUP_VALUE,
  anyOf,
  anyOfIncludes,
  compareVersionLists,
  describeRoles,
  environmentLabel,
} from '../../FleetClustersTab.utils';
import { ClusterHealthCell } from '../cluster-health-cell';
import { ClusterTypeCell } from '../cluster-type-cell';
import { ClusterVersionCell } from '../cluster-version-cell';
import { ThresholdValue } from '../threshold-value';
import { UpdateAvailableChip } from '../update-available-chip';

const { columns, columnTooltips, thresholds } = Messages;

/** A header with its definition on hover. */
const withTooltip = (label: string, tooltip: string) => () => (
  <Tooltip title={tooltip}>
    <span>{label}</span>
  </Tooltip>
);

export const DEFAULT_COLUMN: Partial<MRT_ColumnDef<OmClusterRow>> = {
  enableGlobalFilter: false,
  // The group header names the group; its row repeats it nowhere else.
  GroupedCell: () => null,
  // TanStack blanks a grouped column's cells on the cluster rows; draw them anyway.
  PlaceholderCell: ({ cell, column, row, table }) =>
    column.columnDef.Cell?.({
      cell,
      column,
      row,
      table,
      renderedCellValue: cell.renderValue() as ReactNode,
    }) ?? (cell.renderValue() as ReactNode),
};

/** In the design's order; the last four are in the chooser, hidden at first. */
export const CLUSTER_COLUMNS: MRT_ColumnDef<OmClusterRow>[] = [
  {
    accessorKey: 'cluster_name',
    header: columns.cluster,
    size: 200,
    // The only column the search reads.
    enableGlobalFilter: true,
    // Plain text until it has somewhere to lead: <ClusterNameLink />, PMM-15879.
    Cell: ({ row: { original } }) =>
      original.cluster_name ?? <Unavailable reason="not_applicable" />,
  },
  {
    // The raw type, which sorts as its label does.
    accessorFn: GROUP_VALUE.type,
    id: 'type',
    header: columns.type,
    size: 150,
    filterFn: anyOf((row) => TYPE_FILTER_VALUE[row.type]),
    Cell: ({ row: { original } }) => <ClusterTypeCell cluster={original} />,
  },
  {
    accessorKey: 'health',
    header: columns.health,
    size: 200,
    filterFn: anyOf((row) => row.health),
    // Worst first ascending
    sortingFn: (a, b, columnId) =>
      clusterHealthRank(a.getValue<OmClusterHealth>(columnId)) -
      clusterHealthRank(b.getValue<OmClusterHealth>(columnId)),
    Cell: ({ row: { original } }) => <ClusterHealthCell cluster={original} />,
  },
  {
    accessorFn: (row) => row.up_services,
    id: 'members',
    header: columns.members,
    size: 110,
    Header: withTooltip(columns.members, columnTooltips.members),
    Cell: ({ row: { original } }) =>
      `${original.up_services}/${original.total_services}`,
  },
  {
    accessorFn: GROUP_VALUE.versions,
    id: 'versions',
    header: columns.versions,
    size: 160,
    filterFn: anyOfIncludes((row) => row.versions),
    // As versions, not strings: 7.0.9-1 is older than 7.0.24-13.
    // A cluster with no version known goes after the rest.
    sortingFn: ({ original: a }, { original: b }) =>
      Number(!a.versions.length) - Number(!b.versions.length) ||
      compareVersionLists(a.versions, b.versions),
    Cell: ({ row: { original } }) => (
      <Stack direction="row" alignItems="center" gap={1}>
        <ClusterVersionCell cluster={original} />
        <UpdateAvailableChip available={false} />
      </Stack>
    ),
  },
  {
    accessorKey: 'max_replication_lag_seconds',
    header: columns.maxReplicationLag,
    size: 130,
    Header: withTooltip(
      columns.maxReplicationLag,
      columnTooltips.maxReplicationLag
    ),
    Cell: ({ row: { original } }) => {
      const seconds = original.max_replication_lag_seconds;
      return (
        <ThresholdValue
          seconds={seconds}
          crossed={seconds != null && seconds >= LAG_ALERT_SECONDS}
          level="error"
          reason={thresholds.lagAbove(formatCompactDuration(LAG_ALERT_SECONDS))}
        />
      );
    },
  },
  {
    accessorKey: 'min_oplog_window_seconds',
    header: columns.minOplogWindow,
    size: 130,
    Header: withTooltip(columns.minOplogWindow, columnTooltips.minOplogWindow),
    Cell: ({ row: { original } }) => {
      const seconds = original.min_oplog_window_seconds;
      return (
        <ThresholdValue
          seconds={seconds}
          crossed={seconds != null && seconds < OPLOG_WARN_SECONDS}
          level="warning"
          reason={thresholds.oplogBelow(
            formatCompactDuration(OPLOG_WARN_SECONDS)
          )}
        />
      );
    },
  },
  {
    accessorFn: GROUP_VALUE.env_name,
    id: 'env_name',
    header: columns.environment,
    size: 150,
    filterFn: anyOf((row) => row.env_name || UNNAMED_ENVIRONMENT_VALUE),
    Cell: ({ row: { original } }) => environmentLabel(original),
  },
  {
    accessorFn: (row) =>
      row.last_seen_at ? Date.parse(row.last_seen_at) : undefined,
    id: 'last_seen_at',
    header: columns.lastSeen,
    size: 110,
    sortUndefined: 'last',
    Header: withTooltip(columns.lastSeen, columnTooltips.lastSeen),
    Cell: ({ row: { original } }) =>
      original.last_seen_at ? (
        <Age value={original.last_seen_at} />
      ) : (
        <Unavailable reason="service_not_observed" />
      ),
  },
  { accessorKey: 'total_services', header: columns.services, size: 100 },
  {
    accessorFn: (row) => describeRoles(row.by_process_role),
    id: 'roles',
    header: columns.process,
    size: 130,
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
    Header: withTooltip(
      columns.minConnectionsFree,
      columnTooltips.minConnectionsFree
    ),
    Cell: ({ row: { original } }) =>
      original.min_connections_free_percent == null ? (
        <Unavailable reason="metric_not_collected" />
      ) : (
        <Percent value={original.min_connections_free_percent} />
      ),
  },
];
