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

import { Link as RouterLink } from 'react-router-dom';
import Link from '@mui/material/Link';
import { clusterPath } from '../../../constants';
import type { OmClusterRow } from '../../../types';
import { useOmBase } from '../../../useOmBase';
import { clusterDisplayName } from '../../FleetClustersTab.utils';

/** A cluster's name leading to its page; an unlabelled cluster still has one. */
export const ClusterNameLink = ({ cluster }: { cluster: OmClusterRow }) => {
  const base = useOmBase();
  return (
    <Link
      component={RouterLink}
      to={clusterPath(base, cluster.id)}
      underline="hover"
      fontWeight={500}
      color={cluster.cluster_name ? undefined : 'text.secondary'}
    >
      {clusterDisplayName(cluster)}
    </Link>
  );
};
