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
import { useScanInFlight } from '../inventoryHooks';
import { describeScanProgress } from './ScanProgress.utils';

/**
 * How far the scan in flight has got, beside the control that starts one.
 *
 * Nodes done of nodes to scan, the time so far, and how long one usually takes, so a
 * scan of tens of seconds reads as progressing rather than as a spinner. Nothing
 * while no scan runs.
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
  return (
    <Typography
      variant="body2"
      color="text.secondary"
      data-testid="om-scan-progress"
    >
      {describeScanProgress(run, expectedSeconds, now)}
    </Typography>
  );
};
