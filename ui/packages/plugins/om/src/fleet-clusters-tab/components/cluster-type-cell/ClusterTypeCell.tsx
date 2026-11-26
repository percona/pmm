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
import { Unavailable } from '../../../components/Unavailable';
import type { OmClusterRow } from '../../../types';
import { typeLabel } from '../../FleetClustersTab.utils';
import { TYPE_ICON } from './ClusterTypeCell.constants';

export const ClusterTypeCell = ({ cluster }: { cluster: OmClusterRow }) => {
  const label = typeLabel(cluster);
  if (!label) {
    return <Unavailable reason="not_applicable" />;
  }
  return (
    <Stack
      direction="row"
      alignItems="center"
      gap={0.75}
      sx={{ color: 'text.secondary', '& > span': { color: 'text.primary' } }}
      data-testid="om-cluster-type"
    >
      {TYPE_ICON[cluster.type]}
      <span>{label}</span>
    </Stack>
  );
};
