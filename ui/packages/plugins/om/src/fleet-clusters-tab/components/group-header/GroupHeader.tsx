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
import { ClusterHealthBadge } from '../../../components/HealthBadge';
import { pluralize } from '../../../format';
import type { OmClusterRow } from '../../../types';
import type { ClusterGrouping } from '../../FleetClustersTab.types';
import { environmentLabel } from '../../FleetClustersTab.utils';

export const GroupHeader = ({
  row,
  table,
  staticRowIndex,
  grouping,
}: {
  row: MRT_Row<OmClusterRow>;
  table: MRT_TableInstance<OmClusterRow>;
  staticRowIndex?: number;
  grouping: ClusterGrouping;
}) => {
  const clusters = row.getLeafRows().map((leaf) => leaf.original);
  const sum = (field: 'total_services' | 'up_services' | 'down_services') =>
    clusters.reduce((total, cluster) => total + cluster[field], 0);
  const services = sum('total_services');
  const down = sum('down_services');

  return (
    <Stack
      direction="row"
      alignItems="center"
      gap={2}
      // Its own width, not the expand column's, or the counts collapse to nothing.
      sx={{ width: 'max-content' }}
      data-testid="om-cluster-group"
    >
      <Stack direction="row" alignItems="center" gap={0.5}>
        <MRT_ExpandButton
          row={row}
          table={table}
          staticRowIndex={staticRowIndex}
        />
        {grouping === 'health' ? (
          <ClusterHealthBadge health={row.original.health} />
        ) : (
          <Typography variant="subtitle2" component="h2">
            {environmentLabel(row.original)}
          </Typography>
        )}
      </Stack>
      <Typography variant="body2" color="text.secondary">
        <strong>{clusters.length}</strong>{' '}
        {pluralize(clusters.length, 'cluster')}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        <strong>{services}</strong> {pluralize(services, 'service')}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        <strong>{sum('up_services')}</strong> up
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
