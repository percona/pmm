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

import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { isRunningStatus, useLatestTaskRun } from '../../hooks/useTaskHistory';
import { useElapsedSeconds } from '../../hooks/useElapsedSeconds';
import { formatDuration } from '../../utils/formatDuration';
import { TaskHistoryStatusBadge } from '../TaskHistoryTable';
import { formatAbsoluteTime } from '../ScheduledTasksPanel/periods';
import { firstLine, runFailureReason } from './runFailureReason';

export interface LastRunCardProps {
  /** Task name(s) whose most recent run this block describes. */
  taskNames: string | string[];
  /**
   * Open the execution detail for the run this block describes.
   *
   * Takes no argument on purpose: the caller opens the detail by task name so
   * it keeps resolving the newest run, rather than freezing the row this card
   * happened to be holding when the button was pressed.
   */
  onOpenRun: () => void;
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <Box>
      <Typography variant="caption" color="text.secondary" component="div">
        {label}
      </Typography>
      <Typography variant="body2" component="div">
        {value}
      </Typography>
    </Box>
  );
}

/**
 * A persistent summary of a task's most recent run, for the detail Overview.
 *
 * The Overview used to say nothing at all about the last run: the only failure
 * signal was a banner riding router state, which vanished on reload and left
 * no error text anywhere on the page. This block is fetched, not passed in, so
 * it survives a reload and a DBA arriving the morning after a failed nightly
 * backup still sees that it failed and when.
 *
 * A failed run renders at error severity with the first line of its reason;
 * multi-line reasons and everything else live one click away in the execution
 * detail. While a run is in flight the duration reads as a live elapsed time.
 */
export function LastRunCard({ taskNames, onOpenRun }: LastRunCardProps) {
  const { entry, isLoading, error } = useLatestTaskRun(taskNames);

  const running = entry ? isRunningStatus(entry.status) : false;
  const elapsed = useElapsedSeconds(entry?.started_at, running);
  const reason = firstLine(runFailureReason(entry));

  return (
    <Paper variant="outlined" sx={{ p: 2, mb: 2 }} data-testid="last-run-card">
      <Stack
        direction="row"
        alignItems="center"
        gap={1}
        sx={{ mb: entry || isLoading || error ? 1.5 : 0 }}
      >
        <Typography variant="h6" sx={{ flex: 1 }}>
          Last run
        </Typography>
        {entry && <TaskHistoryStatusBadge status={entry.status} />}
      </Stack>

      {isLoading && <Skeleton variant="rounded" height={56} />}

      {!isLoading && error && (
        <Alert severity="error" data-testid="last-run-card-error">
          Failed to load the last run: {error.message}
        </Alert>
      )}

      {!isLoading && !error && !entry && (
        <Typography
          variant="body2"
          color="text.secondary"
          data-testid="last-run-card-empty"
        >
          This task has not run yet.
        </Typography>
      )}

      {entry && (
        <Stack gap={1.5}>
          <Box
            sx={{
              display: 'grid',
              gap: 2,
              gridTemplateColumns: { xs: '1fr 1fr', sm: 'repeat(3, 1fr)' },
            }}
          >
            <Field
              label="Started"
              value={formatAbsoluteTime(entry.started_at)}
            />
            <Field
              label={running ? 'Elapsed' : 'Duration'}
              value={formatDuration(running ? elapsed : entry.duration)}
            />
            <Field
              label="Finished"
              value={formatAbsoluteTime(entry.finished_at)}
            />
          </Box>

          {entry.status === 'failed' && (
            <Alert severity="error" data-testid="last-run-card-failure">
              {reason ??
                'The reason is not reported with the run — open the log to see what went wrong.'}
            </Alert>
          )}

          <Box>
            <Button
              size="small"
              variant="outlined"
              // Called with no argument, not handed the click event: the
              // prop is typed as taking none.
              onClick={() => onOpenRun()}
              data-testid="last-run-card-view"
            >
              {entry.has_logs || running ? 'View log' : 'View run details'}
            </Button>
          </Box>
        </Stack>
      )}
    </Paper>
  );
}
