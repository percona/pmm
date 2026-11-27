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
import DnsOutlinedIcon from '@mui/icons-material/DnsOutlined';
import type { MRT_ColumnDef } from 'material-react-table';
import { StatusBadge } from '../../../components/HealthBadge';
import { MemberState } from '../../../components/MemberState';
import {
  OplogWindowValue,
  ReplicationLagValue,
} from '../../../fleet-clusters-tab/components/threshold-value';
import { serviceStatusRank } from '../../../topology';
import type { OmService } from '../../../types';
import { Messages } from '../../ClusterDetails.messages';
import {
  memberGroup,
  memberHost,
  memberLag,
  memberRole,
  roleRank,
} from '../../ClusterDetails.utils';

const { columns } = Messages.members;

export const MEMBER_COLUMNS: MRT_ColumnDef<OmService>[] = [
  {
    accessorFn: (service) => memberGroup(service).key,
    id: 'group',
    header: '',
  },
  {
    accessorFn: memberHost,
    id: 'host',
    header: columns.hostname,
    size: 220,
    Cell: ({ row: { original } }) => (
      <Stack
        direction="row"
        alignItems="center"
        gap={1}
        title={original.service_name}
        sx={{ minWidth: 0 }}
      >
        <DnsOutlinedIcon fontSize="small" sx={{ color: 'text.secondary' }} />
        <span>{memberHost(original)}</span>
      </Stack>
    ),
  },
  {
    accessorFn: roleRank,
    id: 'role',
    header: columns.role,
    size: 120,
    Cell: ({ row: { original } }) => {
      const role = memberRole(original);
      return role ? Messages.roles[role] : <MemberState service={original} />;
    },
  },
  {
    accessorFn: (service) => serviceStatusRank(service.status),
    id: 'status',
    header: columns.health,
    size: 220,
    Cell: ({ row: { original } }) => (
      <StatusBadge status={original.status} lastUpAt={original.last_up_at} />
    ),
  },
  {
    accessorFn: (service) => service.oplog_window_seconds ?? undefined,
    id: 'oplog_window_seconds',
    header: columns.oplogWindow,
    size: 140,
    sortUndefined: 'last',
    Cell: ({ row: { original } }) => (
      <OplogWindowValue seconds={original.oplog_window_seconds} />
    ),
  },
  {
    accessorFn: (service) => memberLag(service) ?? undefined,
    id: 'replication_lag_seconds',
    header: columns.replicationLag,
    size: 120,
    sortUndefined: 'last',
    Cell: ({ row: { original } }) => (
      <ReplicationLagValue seconds={memberLag(original)} />
    ),
  },
];

/** Built once, so the table's grouping and visibility memos hold across renders. */
export const GROUPED = ['group'];
export const UNGROUPED: string[] = [];
export const HIDE_GROUP_COLUMN = { group: false };
