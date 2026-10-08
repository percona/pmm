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
import Chip from '@mui/material/Chip';
import { formatAge, formatTimestamp } from '../format';
import { useLastScanFinishedAt } from '../inventoryHooks';
import type { OmTopologySnapshotEnvelope } from '../types';
import { describeLastScan } from './SnapshotBar.utils';

/**
 * How current the page is, on screen rather than buried.
 *
 * OM serves the newest *terminal* run's snapshot, including one that reached no
 * node at all. Without the age and the stale flag in view, a page of em-dashes
 * looks like a broken UI instead of an old or failed discovery.
 *
 * Two lines, one source each: when this data was assembled, and when every node was
 * last scanned. The second is read from the scan history, not from the newest reading
 * in the data, which a live exporter keeps seconds old whatever the scans are doing.
 */
export const SnapshotBar = ({
  envelope,
}: {
  envelope: OmTopologySnapshotEnvelope;
}) => {
  const lastScan = describeLastScan(useLastScanFinishedAt());
  return (
    <Stack direction="row" alignItems="center" gap={1} flexWrap="wrap">
      <Tooltip
        title={`Assembled ${formatTimestamp(envelope.generated_at)}. ${
          envelope.observed_at
            ? `Newest metric or scan reading in it: ${formatTimestamp(envelope.observed_at)}.`
            : 'It holds no metric or scan reading yet.'
        }`}
      >
        <Typography variant="body2" color="text.secondary">
          Data from {formatAge(envelope.generated_at)}
        </Typography>
      </Tooltip>
      <Typography variant="body2" color="text.secondary">
        ·
      </Typography>
      <Tooltip title={lastScan.tooltip}>
        <Typography variant="body2" color="text.secondary">
          {lastScan.label}
        </Typography>
      </Tooltip>
      {envelope.stale && (
        <Chip size="small" color="warning" variant="outlined" label="Stale" />
      )}
    </Stack>
  );
};
