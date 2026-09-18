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

import Chip from '@mui/material/Chip';
import {
  TaskHistoryStatusBadge,
  isTaskHistoryStatus,
} from '../TaskHistoryTable';
import type { PeriodicTaskResponse } from './hooks';

export interface LastRunStatusProps {
  /**
   * The task's last-run result, as reported by the periodic-task API. See the
   * note on {@link LastRunStatus} for how `null`/`undefined` is disambiguated.
   */
  status: PeriodicTaskResponse['last_run_status'];
  /**
   * The task's last-run timestamp. Used only to tell "never run" apart from
   * "ran, but the result could not be resolved" when `status` is absent.
   */
  lastRunAt: PeriodicTaskResponse['last_run_at'];
}

const mutedChipSx = { color: 'text.disabled', borderColor: 'divider' } as const;

/**
 * Shared last-run outcome renderer for the schedule surfaces (list-view cell,
 * detail summary, scheduled-tasks panel row).
 *
 * Renders the outcome with the app-wide {@link TaskHistoryStatusBadge} so
 * colors and labels match execution history. The status enum has no "never
 * run" member, so the empty state is rendered outside the badge.
 *
 * The backend leaves `last_run_status` null in two distinct situations: the
 * schedule has genuinely never run (`last_run_at` is also null), or it ran but
 * no matching execution-history point could be resolved (`last_run_at` is set).
 * These are disambiguated with `lastRunAt` so a task that has run is never
 * mislabeled "Not yet run" next to a real timestamp. A present-but-unrecognized
 * status is shown verbatim rather than dropped, matching the fallback used by
 * the list/detail status columns.
 */
export function LastRunStatus({ status, lastRunAt }: LastRunStatusProps) {
  if (isTaskHistoryStatus(status)) {
    return <TaskHistoryStatusBadge status={status} />;
  }

  if (status !== null && status !== undefined) {
    return (
      <Chip
        label={String(status)}
        size="small"
        variant="outlined"
        data-testid="last-run-unrecognized"
        sx={mutedChipSx}
      />
    );
  }

  const neverRun = lastRunAt === null || lastRunAt === undefined;
  return (
    <Chip
      label={neverRun ? 'Not yet run' : 'Unknown'}
      size="small"
      variant="outlined"
      data-testid={neverRun ? 'last-run-never' : 'last-run-unknown'}
      sx={mutedChipSx}
    />
  );
}
