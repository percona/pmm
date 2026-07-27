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

import { Link as RouterLink } from 'react-router-dom';
import ScheduleIcon from '@mui/icons-material/Schedule';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import {
  describePeriod,
  formatAbsoluteTime,
  formatRelativeTime,
  LastRunStatus,
  selectSchedule,
  useScheduledTasksForPlugin,
} from '../ScheduledTasksPanel';

export interface ScheduleSummaryProps {
  /** Plugin owning the task; used to scope the periodic-task lookup. */
  pluginName: string;
  /** The task's name; joined client-side against the periodic-task list. */
  taskName: string;
  /** Route to the plugin's Schedules screen (the add-a-schedule target). */
  scheduleHref: string;
  /** Disable list polling. Used by stories/tests. */
  disablePolling?: boolean;
}

/**
 * Generic detail-page schedule summary.
 *
 * Shows the next run (relative, with the absolute timestamp on hover) and the
 * recurrence in plain language for a scheduled task, or a "Not scheduled" state
 * with a link to the Schedules screen when the task has no periodic schedule.
 * Framework component shared by any scheduling-capable plugin; it is not
 * archives-specific. The task-to-schedule join is performed client-side by
 * task name.
 */
export function ScheduleSummary({
  pluginName,
  taskName,
  scheduleHref,
  disablePolling = false,
}: ScheduleSummaryProps) {
  const { periodicTasks, isLoading } = useScheduledTasksForPlugin(pluginName, {
    disablePolling,
  });
  const task = selectSchedule(periodicTasks.filter((p) => p.task === taskName));

  return (
    <Paper
      variant="outlined"
      sx={{ p: 3, mb: 3 }}
      data-testid="schedule-summary"
    >
      <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 2 }}>
        <ScheduleIcon fontSize="small" />
        <Typography variant="h6">Schedule</Typography>
      </Stack>

      {isLoading ? (
        <Typography variant="body2" color="text.secondary">
          Loading schedule…
        </Typography>
      ) : task ? (
        <Stack spacing={2} data-testid="schedule-summary-scheduled">
          <Box>
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{ display: 'block', mb: 0.5 }}
            >
              Next run
            </Typography>
            {task.next_run_at ? (
              <Tooltip title={formatAbsoluteTime(task.next_run_at)}>
                <Typography variant="body1" component="span">
                  {formatRelativeTime(task.next_run_at)}
                </Typography>
              </Tooltip>
            ) : (
              <Typography variant="body1" color="text.secondary">
                —
              </Typography>
            )}
          </Box>
          <Box>
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{ display: 'block', mb: 0.5 }}
            >
              Recurrence
            </Typography>
            <Typography variant="body1">
              {describePeriod(task).display}
            </Typography>
          </Box>
          <Box data-testid="schedule-summary-last-run">
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{ display: 'block', mb: 0.5 }}
            >
              Last run
            </Typography>
            <LastRunStatus
              status={task.last_run_status}
              lastRunAt={task.last_run_at}
            />
          </Box>
        </Stack>
      ) : (
        <Stack
          spacing={1.5}
          alignItems="flex-start"
          data-testid="schedule-summary-unscheduled"
        >
          <Typography variant="body2" color="text.secondary">
            Not scheduled
          </Typography>
          <Button
            component={RouterLink}
            to={scheduleHref}
            size="small"
            startIcon={<ScheduleIcon />}
            data-testid="schedule-summary-add-link"
          >
            Add a schedule
          </Button>
        </Stack>
      )}
    </Paper>
  );
}
