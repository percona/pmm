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
import Typography from '@mui/material/Typography';
import { toClusterRows } from '../../../topology';
import { useOmTopology } from '../../../topologyHooks';
import { Messages } from '../../FleetClustersTab.messages';

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
  const members = rows.reduce((sum, row) => sum + row.total_services, 0);
  return (
    <Typography
      variant="body2"
      color="text.secondary"
      data-testid="om-fleet-summary"
    >
      {Messages.summary.clusters(rows.length)} ·{' '}
      {Messages.summary.members(members)}
    </Typography>
  );
};
