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

import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import type { OmClusterRow } from '../../../types';
import { Messages } from '../../FleetClustersTab.messages';

/** Members up out of all; red while one is down, `?` while none has reported. */
export const ClusterMembersCell = ({ cluster }: { cluster: OmClusterRow }) => {
  const {
    total_services: total,
    up_services: up,
    down_services: down,
  } = cluster;

  if (total > 0 && up + down === 0) {
    return (
      <Tooltip title={Messages.membersUnreportedTooltip}>
        <Typography variant="inherit" component="span" tabIndex={0}>
          {Messages.membersUnreported}/{total}
        </Typography>
      </Tooltip>
    );
  }
  return (
    <Typography
      variant="inherit"
      component="span"
      sx={down > 0 ? { color: 'error.main' } : undefined}
    >
      {up}/{total}
    </Typography>
  );
};
