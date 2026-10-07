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

import { useEffect, useState } from 'react';
import { Typography } from '@mui/material';
import { formatCompactDuration, pluralize } from '../format';
import { useScanInFlight } from '../inventoryHooks';

/**
 * How far the scan in flight has got, beside the control that starts one.
 *
 * Nodes done of nodes to scan, the time so far, and how long one usually takes, so a
 * scan of tens of seconds reads as progressing rather than as a spinner. Nothing
 * while no scan runs. The node count needs a server that reports it; without one the
 * time is still shown.
 */
export const ScanProgress = () => {
  const { run, expectedSeconds } = useScanInFlight();
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!run) {
      return;
    }
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, [run]);

  if (!run) {
    return null;
  }
  const elapsed = Math.max(0, (now - Date.parse(run.start_time)) / 1000);
  const total = run.counts.probeable_hosts;
  const parts = [
    total > 0 && run.counts.finished_hosts != null
      ? `${run.counts.finished_hosts} of ${total} ${pluralize(total, 'node')} done`
      : null,
    `${formatCompactDuration(elapsed) || '0s'} so far`,
    expectedSeconds != null
      ? `usually about ${formatCompactDuration(expectedSeconds) || '1s'}`
      : null,
  ].filter((part) => part !== null);
  return (
    <Typography
      variant="body2"
      color="text.secondary"
      data-testid="om-scan-progress"
    >
      {parts.join(' · ')}
    </Typography>
  );
};
