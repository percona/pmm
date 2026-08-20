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

import { Stack, Tooltip, Typography } from '@mui/material';
import { Chip } from '@percona/percona-ui';
import { formatAge, formatTimestamp } from '../format';
import type { OmTopologySnapshotEnvelope } from '../types';

/**
 * Snapshot provenance, on screen rather than buried.
 *
 * OM serves the newest *terminal* run's snapshot, including one that reached no
 * node at all. Without the age and the stale flag in view, a page of em-dashes
 * looks like a broken UI instead of an old or failed discovery.
 */
export function SnapshotBar({
  envelope,
}: {
  envelope: OmTopologySnapshotEnvelope;
}) {
  return (
    <Stack direction="row" alignItems="center" gap={1} flexWrap="wrap">
      <Tooltip
        title={`Snapshot generated ${formatTimestamp(envelope.generated_at)}`}
      >
        <Typography variant="body2" color="text.secondary">
          Snapshot {formatAge(envelope.generated_at)}
        </Typography>
      </Tooltip>
      <Typography variant="body2" color="text.secondary">
        ·
      </Typography>
      <Tooltip
        title={
          envelope.observed_at
            ? `Newest observation ${formatTimestamp(envelope.observed_at)}`
            : 'Nothing in this snapshot was observed'
        }
      >
        <Typography variant="body2" color="text.secondary">
          last observation{' '}
          {envelope.observed_at ? formatAge(envelope.observed_at) : '—'}
        </Typography>
      </Tooltip>
      {envelope.stale && (
        <Chip size="small" color="warning" variant="outlined" label="Stale" />
      )}
    </Stack>
  );
}
