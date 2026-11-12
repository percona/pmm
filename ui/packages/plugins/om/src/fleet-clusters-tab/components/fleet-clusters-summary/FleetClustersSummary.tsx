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
import Divider from '@mui/material/Divider';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { CLUSTER_HEALTH_LABEL } from '../../../constants';
import { toClusterRows } from '../../../topology';
import { useOmTopology } from '../../../topologyHooks';
import { HEALTH_FILTER_VALUES } from '../../FleetClustersTab.constants';
import { Messages } from '../../FleetClustersTab.messages';
import { countValues } from '../../FleetClustersTab.utils';
import { ClusterHealthIcon } from '../cluster-health-icon';

/**
 * The whole fleet in a line, for the page header: how many clusters, and how many in
 * each state, worst first. Unfiltered; the Group by bar counts the matches.
 */
export const FleetClustersSummary = ({
  fallback,
}: {
  /** Shown until the fleet has loaded, or while it holds no cluster. */
  fallback: React.ReactNode;
}) => {
  const { data } = useOmTopology();
  const rows = useMemo(() => toClusterRows(data), [data]);
  if (!rows.length) {
    return fallback;
  }
  const byHealth = countValues(rows, (row) => row.health);
  return (
    <Stack
      direction="row"
      alignItems="center"
      gap={1.5}
      flexWrap="wrap"
      data-testid="om-fleet-summary"
    >
      <Typography variant="body2" color="text.secondary">
        {Messages.summary.clusters(rows.length)}
      </Typography>
      <Divider orientation="vertical" flexItem />
      {HEALTH_FILTER_VALUES.filter((health) => byHealth[health]).map(
        (health) => (
          <Stack key={health} direction="row" alignItems="center" gap={0.5}>
            <ClusterHealthIcon health={health} />
            <Typography variant="body2" color="text.secondary">
              {Messages.summary.health(
                byHealth[health],
                CLUSTER_HEALTH_LABEL[health]
              )}
            </Typography>
          </Stack>
        )
      )}
    </Stack>
  );
};
