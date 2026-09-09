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

import RepeatIcon from '@mui/icons-material/Repeat';
import Chip from '@mui/material/Chip';
import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import {
  describePeriod,
  formatAbsoluteTime,
  formatRelativeTime,
  LastRunStatus,
  type PeriodicTaskResponse,
} from '../ScheduledTasksPanel';

export interface ScheduleCellProps {
  /**
   * The periodic task matched to this row by name, or `null`/`undefined` when
   * the row's task has no schedule.
   */
  task?: PeriodicTaskResponse | null;
  /**
   * Whether the schedule list is still loading. When loading and no task has
   * matched yet, a placeholder is shown instead of the "Not scheduled" chip so
   * rows do not flash unscheduled before the schedule data arrives.
   */
  isLoading?: boolean;
}

/**
 * Generic list-view schedule cell.
 *
 * Shows the next run as relative time (with the absolute timestamp on hover)
 * plus a periodicity icon whose tooltip describes the recurrence in plain
 * language. Renders a muted "Not scheduled" chip when the row's task has no
 * periodic schedule. Framework component shared by any plugin that declares a
 * `schedule`-format column; it is not archives-specific.
 */
export function ScheduleCell({ task, isLoading = false }: ScheduleCellProps) {
  if (!task) {
    if (isLoading) {
      return (
        <Skeleton
          variant="text"
          width={72}
          data-testid="schedule-cell-loading"
        />
      );
    }
    return (
      <Chip
        label="Not scheduled"
        size="small"
        variant="outlined"
        data-testid="schedule-cell-unscheduled"
        sx={{ color: 'text.disabled', borderColor: 'divider' }}
      />
    );
  }

  const period = describePeriod(task);
  const nextRun = task.next_run_at;

  return (
    <Stack
      direction="row"
      spacing={0.5}
      alignItems="center"
      data-testid="schedule-cell"
    >
      {nextRun ? (
        <Tooltip title={formatAbsoluteTime(nextRun)}>
          <Typography
            variant="body2"
            component="span"
            data-testid="schedule-cell-next-run"
          >
            {formatRelativeTime(nextRun)}
          </Typography>
        </Tooltip>
      ) : (
        <Typography
          variant="body2"
          component="span"
          color="text.secondary"
          data-testid="schedule-cell-next-run"
        >
          —
        </Typography>
      )}
      <Tooltip
        title={
          period.tooltip
            ? `${period.display} (${period.tooltip})`
            : period.display
        }
      >
        <RepeatIcon
          fontSize="inherit"
          aria-label={`Recurs ${period.display}`}
          data-testid="schedule-cell-periodicity"
          sx={{ color: 'text.secondary' }}
        />
      </Tooltip>
      <LastRunStatus
        status={task.last_run_status}
        lastRunAt={task.last_run_at}
      />
    </Stack>
  );
}
