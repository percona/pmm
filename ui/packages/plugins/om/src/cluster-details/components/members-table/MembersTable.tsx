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
import { Link as RouterLink } from 'react-router-dom';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/peak-ui';
import { MRT_ExpandButton } from 'material-react-table';
import { RowOverflowMenu } from '../../../components/RowOverflowMenu';
import { serviceDashboardPath } from '../../../components/ServiceLink';
import type { OmClusterRow, OmService } from '../../../types';
import { useTableUrlState } from '../../../useTableUrlState';
import { availableMemberActions } from '../../ClusterDetails.actions';
import {
  MEMBERS_TABLE_NAME,
  MEMBERS_URL_STATE,
} from '../../ClusterDetails.constants';
import { Messages } from '../../ClusterDetails.messages';
import {
  compareMembers,
  memberGroup,
  memberHost,
  memberRole,
} from '../../ClusterDetails.utils';
import {
  GROUPED,
  HIDE_GROUP_COLUMN,
  MEMBER_COLUMNS,
  UNGROUPED,
} from './MembersTable.constants';

const MembersSummary = ({ cluster }: { cluster: OmClusterRow }) => (
  <Stack direction="row" alignItems="baseline" gap={1.5} flexWrap="wrap">
    <Typography variant="h6" component="h2">
      {Messages.members.title}
    </Typography>
    <Typography
      variant="body2"
      color="text.secondary"
      data-testid="om-members-reachable"
    >
      {Messages.members.reachable(cluster.up_services, cluster.total_services)}
    </Typography>
    <Typography
      variant="body2"
      color={cluster.down_services ? 'error.main' : 'success.main'}
    >
      {cluster.down_services
        ? Messages.members.down(cluster.down_services)
        : Messages.members.allHealthy}
    </Typography>
  </Stack>
);

const MemberActions = ({ service }: { service: OmService }) => (
  <RowOverflowMenu label={Messages.members.actions(memberHost(service))}>
    {(close) => [
      <MenuItem
        key="dashboards"
        component={RouterLink}
        to={serviceDashboardPath(service.service_name)}
        onClick={close}
      >
        {Messages.members.openDashboards}
      </MenuItem>,
      ...availableMemberActions(memberRole(service)).map((action) => (
        <MenuItem key={action.id} onClick={close}>
          {action.label}
        </MenuItem>
      )),
    ]}
  </RowOverflowMenu>
);

export const MembersTable = ({ cluster }: { cluster: OmClusterRow }) => {
  const grouped = cluster.type === 'CLUSTER_TYPE_SHARDED';
  // Groups appear in the order their first row arrives, so sort into group order first.
  const data = useMemo(
    () =>
      [...cluster.services].sort(
        grouped
          ? (a, b) =>
              memberGroup(a).key.localeCompare(memberGroup(b).key) ||
              compareMembers(a, b)
          : compareMembers
      ),
    [cluster.services, grouped]
  );
  const tableProps = useTableUrlState(MEMBERS_URL_STATE);

  return (
    <Paper variant="outlined" sx={{ p: 2 }} data-testid="om-members-section">
      <Stack gap={2}>
        <MembersSummary cluster={cluster} />
        <Table
          {...tableProps}
          tableName={MEMBERS_TABLE_NAME}
          columns={MEMBER_COLUMNS}
          data={data}
          noDataMessage={Messages.members.empty}
          getRowId={(row) => row.service_id ?? row.service_name}
          enablePagination={false}
          enableTopToolbar={false}
          enableBottomToolbar={false}
          enableColumnActions={false}
          enableColumnFilters={false}
          enableGlobalFilter={false}
          enableHiding={false}
          enableGrouping={grouped}
          groupedColumnMode="remove"
          enableExpanding={grouped}
          enableRowActions
          positionActionsColumn="last"
          renderRowActions={({ row }) =>
            row.getIsGrouped() ? null : <MemberActions service={row.original} />
          }
          muiTableBodyRowProps={({ row }) =>
            row.getIsGrouped()
              ? { sx: { backgroundColor: 'action.hover' } }
              : {}
          }
          displayColumnDefOptions={{
            'mrt-row-expand': {
              size: 40,
              Header: () => null,
              Cell: ({ row, table, staticRowIndex }) =>
                row.getIsGrouped() ? (
                  <Stack
                    direction="row"
                    alignItems="center"
                    gap={1}
                    sx={{ width: 'max-content' }}
                    data-testid="om-member-group"
                  >
                    <MRT_ExpandButton
                      row={row}
                      table={table}
                      staticRowIndex={staticRowIndex}
                    />
                    <Typography variant="subtitle2" component="h3">
                      {memberGroup(row.original).label}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {Messages.members.groupCount(row.getLeafRows().length)}
                    </Typography>
                  </Stack>
                ) : null,
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
            grouping: grouped ? GROUPED : UNGROUPED,
            columnVisibility: HIDE_GROUP_COLUMN,
          }}
          initialState={{ density: 'compact', expanded: true }}
        />
      </Stack>
    </Paper>
  );
};
