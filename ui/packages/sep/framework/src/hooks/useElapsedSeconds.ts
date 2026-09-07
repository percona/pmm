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

const TICK_INTERVAL_MS = 1000;

function secondsSince(startedAt: string): number | null {
  const started = new Date(startedAt).getTime();
  if (Number.isNaN(started)) {
    return null;
  }
  return (Date.now() - started) / 1000;
}

/**
 * Seconds elapsed since `startedAt`, re-rendering once a second while
 * `isRunning`.
 *
 * The wire carries no duration for a run in flight — `TaskHistoryResponse.
 * duration` is server-computed from `finished_at` and stays null until the run
 * ends — so an in-flight elapsed time has to be derived here. Nothing else in
 * the framework tickes, so this is the only place a clock runs.
 *
 * The ticker is torn down as soon as `isRunning` goes false, and the final
 * value is left in place rather than zeroed: a caller that swaps to the
 * server's `duration` on the same render never flashes an empty cell, and one
 * that does not keeps showing the last elapsed time it observed.
 *
 * Returns null when there is no usable start time, which the caller should
 * render the same way it renders an absent duration.
 */
export function useElapsedSeconds(
  startedAt: string | null | undefined,
  isRunning: boolean
): number | null {
  const [elapsed, setElapsed] = useState<number | null>(() =>
    startedAt ? secondsSince(startedAt) : null
  );

  useEffect(() => {
    if (!startedAt) {
      setElapsed(null);
      return;
    }

    // Recompute on every dependency change, not only on tick: a run that
    // finishes between ticks, or a drawer reopened on a different run, must
    // show the right number immediately rather than up to a second late.
    setElapsed(secondsSince(startedAt));

    if (!isRunning) {
      return;
    }

    const timer = setInterval(() => {
      setElapsed(secondsSince(startedAt));
    }, TICK_INTERVAL_MS);

    return () => clearInterval(timer);
  }, [startedAt, isRunning]);

  return elapsed;
}
