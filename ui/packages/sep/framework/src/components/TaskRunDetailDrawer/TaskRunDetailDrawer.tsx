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

import CloseIcon from '@mui/icons-material/Close';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Drawer from '@mui/material/Drawer';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import {
  isRunningStatus,
  type TaskHistoryEntry,
  useLatestTaskRun,
} from '../../hooks/useTaskHistory';
import { useElapsedSeconds } from '../../hooks/useElapsedSeconds';
import { formatDuration } from '../../utils/formatDuration';
import { TaskHistoryStatusBadge } from '../TaskHistoryTable';
import { TaskLogViewer } from '../TaskLogViewer';
import { formatAbsoluteTime } from '../ScheduledTasksPanel/periods';
import { runFailureReason } from './runFailureReason';

/**
 * Height handed to the embedded log viewer.
 *
 * A CSS `clamp` rather than a measured pixel value: the viewer applies its
 * height to an inner box, so a percentage would resolve against an
 * indefinite-height parent and collapse the pane to nothing. This keeps the
 * log the tallest element the drawer can afford without a resize observer.
 */
const LOG_HEIGHT = 'clamp(240px, calc(100vh - 460px), 900px)';

const DRAWER_WIDTH = { xs: '100%', sm: 560, md: 760 } as const;

/**
 * Id linking the drawer's heading to its dialog role.
 *
 * A constant rather than a generated id: only one run detail is open at a
 * time — each call site mounts the drawer only while it is showing something.
 */
const HEADING_ID = 'run-detail-heading';

/**
 * Why a terminal run that did not succeed ended, for the statuses whose
 * meaning is not self-evident from the badge alone.
 *
 * `failed` is deliberately absent: it renders at error severity with the
 * backend's own reason (or a pointer to the log), rather than a fixed sentence.
 */
const NON_FAILURE_TERMINAL_NOTES: Partial<Record<string, string>> = {
  stopped: 'This run was stopped before it finished.',
  lost: 'The executor stopped reporting on this run, so its outcome is unknown.',
  stale:
    'This run was skipped because the executor could not place it before the staleness threshold.',
  unlaunchable:
    'The executor node could not launch this run, so the payload never executed. This is not a script failure.',
};

/**
 * Explanation for a run that finished with no readable log.
 *
 * Uses `log_capture`, which distinguishes a run that genuinely produced nothing
 * from one whose output was lost before SEP could read it — a difference that
 * matters when the reader is trying to establish whether a backup did anything.
 */
const NO_LOG_NOTES: Record<string, string> = {
  incomplete:
    'This run produced log output, but it was lost before it could be stored.',
  unknown: 'No log was recorded for this run.',
  complete: 'This run produced no log output.',
};

function SummaryField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <Box>
      <Typography variant="caption" color="text.secondary" component="div">
        {label}
      </Typography>
      <Typography variant="body2" component="div">
        {children}
      </Typography>
    </Box>
  );
}

export interface TaskRunDetailDrawerProps {
  open: boolean;
  onClose: () => void;
  /**
   * A run already in hand. Supplied by execution history, whose rows carry a
   * `task_history_id`; takes precedence over `taskNames`.
   *
   * Rendered exactly as given — the drawer does not re-read it. A caller that
   * shows a run still in flight must therefore pass a row it keeps fresh
   * (execution history re-reads the opened row out of its own polling query),
   * or the status here freezes at whatever it was when the drawer opened.
   */
  entry?: TaskHistoryEntry | null;
  /**
   * Task name(s) whose most recent run should be shown, for the surfaces that
   * hold a name and no run id — the list status chip, the detail Overview and
   * the schedules "Last Run" cell. Resolved only while the drawer is open, so
   * a table of rows costs one request on click rather than one per row.
   */
  taskNames?: string | string[];
  /**
   * With `taskNames`, the run that started at this time rather than the newest
   * one. The schedules cell passes its own `last_run_at`: a task that also runs
   * manually would otherwise open a newer, unrelated run.
   */
  at?: string | null;
  /**
   * A run known only by id, with no history row to describe it — the
   * post-create connectivity check, which reports a `task_history_id` and
   * nothing else. The log is shown without the summary, because there is no
   * status, timing or reason to show. Ignored when `entry` or `taskNames` is
   * given.
   */
  taskHistoryId?: number | string;
  /** Heading fallback for a run whose own payload does not name its task. */
  taskLabel?: string;
}

/**
 * The execution detail for a single task run: status, timing, why it failed,
 * and its log.
 *
 * One view behind every entry point, so a run opened from a status chip, the
 * Overview, execution history or a schedule's last-run cell shows the same
 * thing. It replaces the two log dialogs the detail page used to own, which
 * showed the log and nothing around it.
 *
 * A run in flight gets a live elapsed time and a streaming log: the viewer
 * holds an SSE connection open, so neither needs a reload. Whether the run's
 * *status* keeps up depends on how it was opened — by name it is re-resolved
 * on the poll, and `entry` is only as fresh as the caller keeps it.
 */
export function TaskRunDetailDrawer({
  open,
  onClose,
  entry,
  taskNames,
  at,
  taskHistoryId,
  taskLabel,
}: TaskRunDetailDrawerProps) {
  // Resolve by name only when no row was handed in, and only while open.
  const shouldResolve = open && !entry && taskNames !== undefined;
  const latest = useLatestTaskRun(taskNames, { at, enabled: shouldResolve });

  const run = entry ?? (shouldResolve ? latest.entry : null);
  const isLoading = shouldResolve && latest.isLoading;

  const running = run ? isRunningStatus(run.status) : false;
  const elapsed = useElapsedSeconds(run?.started_at, running);

  const heading =
    run?.display_name || run?.task?.name || taskLabel || 'Task run';
  const failureReason = runFailureReason(run);
  const nonFailureNote = run
    ? NON_FAILURE_TERMINAL_NOTES[run.status]
    : undefined;
  // With no history row there is nothing to gate on, so an id on its own is
  // taken at face value and the log is attempted.
  const logId = run?.id ?? (run ? null : (taskHistoryId ?? null));
  const showLog =
    logId !== null && logId !== undefined && (!run || run.has_logs || running);

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      slotProps={{
        paper: {
          sx: { width: DRAWER_WIDTH },
          // The drawer is a modal detail view, so it is announced as a dialog
          // named by its own heading rather than as an unlabelled region.
          role: 'dialog',
          'aria-modal': true,
          'aria-labelledby': HEADING_ID,
        },
      }}
      data-testid="run-detail-drawer"
    >
      <Stack sx={{ height: '100%' }}>
        <Stack
          direction="row"
          alignItems="center"
          gap={1}
          sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}
        >
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography variant="h6" noWrap title={heading} id={HEADING_ID}>
              {heading}
              {logId !== null && logId !== undefined ? ` #${logId}` : ''}
            </Typography>
          </Box>
          {run && <TaskHistoryStatusBadge status={run.status} />}
          <IconButton
            size="small"
            onClick={onClose}
            aria-label="Close run details"
          >
            <CloseIcon />
          </IconButton>
        </Stack>

        <Box sx={{ p: 2, flex: 1, overflowY: 'auto' }}>
          {isLoading && (
            <Stack alignItems="center" py={4}>
              <CircularProgress data-testid="run-detail-loading" />
            </Stack>
          )}

          {!isLoading && latest.error && !entry && (
            <Alert severity="error" data-testid="run-detail-error">
              Failed to load the latest run: {latest.error.message}
            </Alert>
          )}

          {!isLoading && !latest.error && !run && !showLog && (
            <Alert severity="info" data-testid="run-detail-empty">
              {at
                ? 'No execution history was found for this run.'
                : 'This task has not run yet.'}
            </Alert>
          )}

          {!run && showLog && (
            <TaskLogViewer taskHistoryId={logId} height={LOG_HEIGHT} />
          )}

          {run && (
            <Stack gap={2}>
              <Box
                sx={{
                  display: 'grid',
                  gap: 2,
                  gridTemplateColumns: {
                    xs: '1fr 1fr',
                    sm: 'repeat(3, 1fr)',
                  },
                }}
              >
                <SummaryField label="Started">
                  {formatAbsoluteTime(run.started_at)}
                </SummaryField>
                <SummaryField label={running ? 'Elapsed' : 'Duration'}>
                  <span data-testid="run-detail-duration">
                    {formatDuration(running ? elapsed : run.duration)}
                  </span>
                </SummaryField>
                <SummaryField label="Finished">
                  {formatAbsoluteTime(run.finished_at)}
                </SummaryField>
                {run.executed_by && (
                  <SummaryField label="Executed by">
                    {run.executed_by}
                  </SummaryField>
                )}
              </Box>

              {run.status === 'failed' && (
                <Alert
                  severity="error"
                  sx={{ whiteSpace: 'pre-wrap' }}
                  data-testid="run-detail-failure"
                >
                  <AlertTitle>This run failed</AlertTitle>
                  {failureReason ??
                    'The reason is not reported with the run. The log below is the only record of what went wrong.'}
                </Alert>
              )}

              {nonFailureNote && (
                <Alert severity="warning" data-testid="run-detail-note">
                  {nonFailureNote}
                </Alert>
              )}

              {showLog && logId !== null && logId !== undefined ? (
                <TaskLogViewer
                  taskHistoryId={logId}
                  taskStatus={run.status}
                  height={LOG_HEIGHT}
                />
              ) : (
                <Typography
                  variant="body2"
                  color="text.secondary"
                  data-testid="run-detail-no-log"
                >
                  {NO_LOG_NOTES[run.log_capture] ?? NO_LOG_NOTES.unknown}
                </Typography>
              )}
            </Stack>
          )}
        </Box>
      </Stack>
    </Drawer>
  );
}
