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
import { Table } from '@percona/peak-ui';
import { MRT_ExpandAllButton, MRT_ExpandButton } from 'material-react-table';
import {
  GroupBySelect,
  NO_GROUPING,
  useSessionGrouping,
} from '../../../components/GroupBySelect';
import type { OmClusterRow } from '../../../types';
import { useTableUrlState } from '../../../useTableUrlState';
import {
  CLUSTERS_TABLE_NAME,
  DEFAULT_GROUPING,
  GROUPING_STORAGE_KEY,
  GROUP_BY_OPTIONS,
  URL_STATE,
} from '../../FleetClustersTab.constants';
import { Messages } from '../../FleetClustersTab.messages';
import type { ClusterGrouping } from '../../FleetClustersTab.types';
import { groupLabel } from '../../FleetClustersTab.utils';
import { useClusterColumnVisibility } from '../../useClusterColumnVisibility';
import { useGroupedExpansion } from '../../useGroupedExpansion';
import { ClusterFilterBar } from '../cluster-filter-bar';
import { ClusterMembersTable } from '../cluster-members-table';
import { GroupHeader } from '../group-header';
import { CLUSTER_COLUMNS } from './ClustersTable.constants';

export const ClustersTable = ({ rows }: { rows: OmClusterRow[] }) => {
  const [grouping, setGrouping] = useSessionGrouping<ClusterGrouping>(
    GROUPING_STORAGE_KEY,
    GROUP_BY_OPTIONS,
    DEFAULT_GROUPING
  );
  const tableProps = useTableUrlState(URL_STATE);
  // The whole fleet, not the matches: a folded group stays folded while a filter hides it.
  const { expanded, onExpandedChange } = useGroupedExpansion(rows, grouping);
  const { columnVisibility, onColumnVisibilityChange } =
    useClusterColumnVisibility();

  return (
    <Table
      {...tableProps}
      tableName={CLUSTERS_TABLE_NAME}
      columns={CLUSTER_COLUMNS}
      data={rows}
      enablePagination={false}
      enableExpanding
      enableGrouping
      // The group header already names what the grouped column would repeat on every row.
      groupedColumnMode="remove"
      // Grouping is the select's job: no drag handles, no "grouped by" banner.
      enableColumnDragging={false}
      positionToolbarAlertBanner="none"
      enableColumnActions={false}
      enableColumnFilters={false}
      globalFilterFn="contains"
      defaultColumn={{ enableGlobalFilter: false }}
      // TanStack only searches a column whose first row holds a string; a cluster can lack a name.
      getColumnCanGlobalFilter={() => true}
      muiSearchTextFieldProps={{
        placeholder: Messages.search,
        inputProps: { 'aria-label': Messages.search },
      }}
      enableHiding
      // The server-issued id: labels repeat or are missing, and expansion must survive refetches.
      getRowId={(row) => row.id}
      renderDetailPanel={({ row }) => (
        <ClusterMembersTable cluster={row.original} />
      )}
      emptyFilterResultsMessage={Messages.noMatch}
      muiExpandButtonProps={({ row }) =>
        row.getIsGrouped() && grouping !== NO_GROUPING
          ? {
              'aria-expanded': row.getIsExpanded(),
              'aria-label': Messages.groupToggle(
                row.getIsExpanded(),
                groupLabel(row.original, grouping)
              ),
            }
          : {}
      }
      renderTopToolbarCustomActions={({ table }) => (
        <Stack direction="row" alignItems="center" gap={1.5} flexWrap="wrap">
          <GroupBySelect
            options={GROUP_BY_OPTIONS}
            value={grouping}
            onChange={(value) => setGrouping(value as ClusterGrouping)}
          />
          <ClusterFilterBar table={table} rows={rows} />
        </Stack>
      )}
      displayColumnDefOptions={{
        'mrt-row-expand': {
          size: 60,
          Header: ({ table }) => <MRT_ExpandAllButton table={table} />,
          Cell: ({ row, table, staticRowIndex }) =>
            row.getIsGrouped() ? (
              <GroupHeader
                row={row}
                table={table}
                staticRowIndex={staticRowIndex}
                grouping={grouping}
              />
            ) : (
              <MRT_ExpandButton
                row={row}
                table={table}
                staticRowIndex={staticRowIndex}
              />
            ),
          // A group header runs across the empty cells beside it, so it paints above them.
          muiTableBodyCellProps: ({ row }) =>
            row.getIsGrouped()
              ? {
                  sx: {
                    overflow: 'visible',
                    whiteSpace: 'nowrap',
                    position: 'relative',
                    zIndex: 1,
                  },
                }
              : {},
        },
      }}
      state={{
        ...tableProps.state,
        grouping: grouping === NO_GROUPING ? [] : [grouping],
        expanded,
        columnVisibility,
      }}
      onExpandedChange={onExpandedChange}
      onColumnVisibilityChange={onColumnVisibilityChange}
      initialState={{ density: 'compact' }}
    />
  );
};
