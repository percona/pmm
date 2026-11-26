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
import Typography from '@mui/material/Typography';
import {
  MRT_ShowHideColumnsButton,
  type MRT_TableInstance,
} from 'material-react-table';
import { GroupBySelect, NO_GROUPING } from '../../../components/GroupBySelect';
import type { OmClusterRow } from '../../../types';
import { GROUP_BY_OPTIONS } from '../../FleetClustersTab.constants';
import { Messages } from '../../FleetClustersTab.messages';
import type { ClusterGrouping } from '../../FleetClustersTab.types';
import { RefreshedAgo } from '../refreshed-ago';

/** Group by, what it makes of the matches, how fresh they are, and the column chooser. */
export const GroupByBar = ({
  table,
  grouping,
  onGroupingChange,
  total,
  updatedAt,
}: {
  table: MRT_TableInstance<OmClusterRow>;
  grouping: ClusterGrouping;
  onGroupingChange: (grouping: ClusterGrouping) => void;
  total: number;
  updatedAt: number;
}) => {
  const shown = table.getFilteredRowModel().rows.length;
  return (
    <Stack
      direction="row"
      alignItems="center"
      justifyContent="space-between"
      gap={2}
      flexWrap="wrap"
      sx={{ width: '100%' }}
    >
      <Stack direction="row" alignItems="center" gap={1.5}>
        <GroupBySelect
          options={GROUP_BY_OPTIONS}
          value={grouping}
          onChange={(value) => onGroupingChange(value as ClusterGrouping)}
        />
        <Typography variant="body2" color="text.secondary">
          {grouping !== NO_GROUPING &&
            `${Messages.groupBar.groups(table.getGroupedRowModel().rows.length)} · `}
          <span data-testid="om-cluster-match-count">
            {Messages.groupBar.clusters(shown, total)}
          </span>
        </Typography>
      </Stack>
      <Stack direction="row" alignItems="center" gap={1}>
        {updatedAt > 0 && <RefreshedAgo updatedAt={updatedAt} />}
        <MRT_ShowHideColumnsButton table={table} />
      </Stack>
    </Stack>
  );
};
