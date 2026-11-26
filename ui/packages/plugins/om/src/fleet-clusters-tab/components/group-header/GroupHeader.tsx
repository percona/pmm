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
  MRT_ExpandButton,
  type MRT_Row,
  type MRT_TableInstance,
} from 'material-react-table';
import type { OmClusterRow } from '../../../types';
import type { GroupingColumn } from '../../FleetClustersTab.types';
import { groupLabel } from '../../FleetClustersTab.utils';
import { ClusterHealthIcon } from '../cluster-health-icon';

/** A group's name and how many of the matching clusters it holds. */
export const GroupHeader = ({
  row,
  table,
  staticRowIndex,
  grouping,
}: {
  row: MRT_Row<OmClusterRow>;
  table: MRT_TableInstance<OmClusterRow>;
  staticRowIndex?: number;
  grouping: GroupingColumn;
}) => (
  <Stack
    direction="row"
    alignItems="center"
    gap={1}
    // Its own width, not the expand column's, or the label collapses to nothing.
    sx={{ width: 'max-content' }}
    data-testid="om-cluster-group"
  >
    <MRT_ExpandButton row={row} table={table} staticRowIndex={staticRowIndex} />
    {grouping === 'health' && (
      <ClusterHealthIcon health={row.original.health} />
    )}
    <Typography variant="subtitle2" component="h2">
      {groupLabel(row.original, grouping)}
    </Typography>
    <Typography
      variant="body2"
      color="text.secondary"
      data-testid="om-cluster-group-count"
    >
      {row.getLeafRows().length}
    </Typography>
  </Stack>
);
