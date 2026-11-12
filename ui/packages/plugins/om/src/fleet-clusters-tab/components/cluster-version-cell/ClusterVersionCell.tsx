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

import Box from '@mui/material/Box';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { Unavailable } from '../../../components/Unavailable';
import type { OmClusterRow } from '../../../types';
import { Messages } from '../../FleetClustersTab.messages';

export const ClusterVersionCell = ({ cluster }: { cluster: OmClusterRow }) => {
  if (!cluster.versions.length) {
    return <Unavailable reason="service_not_observed" />;
  }
  if (cluster.version_drift) {
    // A replica set running two versions is a finding, not a longer cell.
    return (
      <Tooltip
        title={
          <Box data-testid="om-version-sets">
            {cluster.version_sets.map((set) => (
              <div key={set.replication_set}>
                {set.replication_set}: {set.versions.join(', ')}
              </div>
            ))}
          </Box>
        }
      >
        <Typography variant="body2" component="span" color="warning.main">
          {Messages.versions.mixed}
        </Typography>
      </Tooltip>
    );
  }
  if (cluster.versions.length === 1) {
    return <>{cluster.versions[0]}</>;
  }
  // Shards or routers on different versions, each set consistent: not drift.
  return (
    <Tooltip title={cluster.versions.join(', ')}>
      <span>{Messages.versions.count(cluster.versions.length)}</span>
    </Tooltip>
  );
};
