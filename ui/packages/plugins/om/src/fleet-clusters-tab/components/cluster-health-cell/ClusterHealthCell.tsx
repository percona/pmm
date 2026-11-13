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
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import type { OmClusterRow } from '../../../types';
import { ClusterHealthBadge } from '../cluster-health-badge';
import { healthReason } from './ClusterHealthCell.utils';

/** The badge, and below it why: the members that are down or silent. */
export const ClusterHealthCell = ({ cluster }: { cluster: OmClusterRow }) => {
  const badge = <ClusterHealthBadge health={cluster.health} />;
  const reason = healthReason(cluster);
  if (!reason) {
    return badge;
  }
  const named = !!reason.members?.length;
  const text = (
    <Typography
      variant="caption"
      color="text.secondary"
      noWrap
      // Focusable when it has a tooltip, so a keyboard reaches the member names.
      tabIndex={named ? 0 : undefined}
      data-testid="om-cluster-health-reason"
    >
      {reason.text}
    </Typography>
  );
  return (
    <Stack alignItems="flex-start" gap={0.25} sx={{ minWidth: 0 }}>
      {badge}
      {named ? (
        <Tooltip title={reason.members?.join(', ')}>{text}</Tooltip>
      ) : (
        text
      )}
    </Stack>
  );
};
