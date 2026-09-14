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

import { useState } from 'react';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { LastRunStatus } from './LastRunStatus';
import { ScheduledTaskForm } from './ScheduledTaskForm';
import { describePeriod } from './periods';
import { RunTime } from '../TaskRunDetailDrawer/RunTime';
import { scheduleTimezone } from './timezones';
import type { AvailableTask } from '../ChainBuilder';
import type {
  PeriodicTaskCreate,
  PeriodicTaskResponse,
  PeriodicTaskUpdate,
} from './hooks';

/**
 * Width of the row, for the edit form's `colSpan`. Must track `columnHeaders`
 * in `ScheduledTasksPanel`: Task, Period, Start Time, Last Run, Next Run, Runs,
 * Enabled, plus Chain and Actions when those are shown.
 */
function columnCount(showChain: boolean, showActions: boolean): number {
  return 7 + (showChain ? 1 : 0) + (showActions ? 1 : 0);
}

export interface ScheduledTaskRowProps {
  task: PeriodicTaskResponse;
  availableTasks: AvailableTask[];
  isEditing: boolean;
  onStartEdit: () => void;
  onCancelEdit: () => void;
  onToggleEnabled: (task: PeriodicTaskResponse, nextEnabled: boolean) => void;
  onSubmitEdit: (
    body: PeriodicTaskCreate | PeriodicTaskUpdate,
    taskName: string
  ) => Promise<void>;
  onDelete: (task: PeriodicTaskResponse) => void;
  submitting?: boolean;
  toggling?: boolean;
  errorMessage?: string;
  /**
   * Hide the enable toggle and the row's edit / delete controls, leaving the
   * schedule readable. Set for sessions that may not mutate.
   */
  readOnly?: boolean;
  /**
   * Open the execution detail for this schedule's last run. Omit to leave the
   * last-run cell inert, as it was before.
   *
   * Carries `last_run_at` as well as the name: the periodic-task API reports no
   * `task_history_id`, so the timestamp is the only thing that identifies which
   * run this row is describing.
   */
  onOpenLastRun?: (taskName: string, lastRunAt: string | null) => void;
  /** Mid-sentence singular noun for one record (e.g. `backup`). */
  itemName?: string;
  /** Mid-sentence plural noun (e.g. `backups`). */
  itemNamePlural?: string;
  /**
   * Render the Chain cell. The panel drops the column while no schedule carries
   * a chain (PMM-15454); the row must drop the matching cell or the table
   * shears.
   */
  showChain?: boolean;
}

export function ScheduledTaskRow({
  task,
  availableTasks,
  isEditing,
  onStartEdit,
  onCancelEdit,
  onToggleEnabled,
  onSubmitEdit,
  onDelete,
  submitting,
  toggling,
  errorMessage,
  readOnly = false,
  onOpenLastRun,
  itemName = 'task',
  itemNamePlural = 'tasks',
  showChain = true,
}: ScheduledTaskRowProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const period = describePeriod(task);
  const chainNames = task.execute_request?.chain_task_names ?? [];
  // The zone the schedule actually fires in, which is not the zone its
  // timestamps are rendered in - the panel names that one once for the table.
  const firesIn = scheduleTimezone(task);

  if (isEditing) {
    return (
      <TableRow>
        <TableCell colSpan={columnCount(showChain, !readOnly)} sx={{ p: 0 }}>
          <ScheduledTaskForm
            mode="edit"
            initialValue={task}
            availableTasks={availableTasks}
            itemName={itemName}
            itemNamePlural={itemNamePlural}
            onCancel={onCancelEdit}
            onSubmit={onSubmitEdit}
            submitting={submitting}
            errorMessage={errorMessage}
          />
        </TableCell>
      </TableRow>
    );
  }

  return (
    <>
      <TableRow data-testid={`scheduled-task-row-${task.id}`}>
        <TableCell>{task.task}</TableCell>
        <TableCell>
          <Stack spacing={0.25} alignItems="flex-start">
            {period.tooltip ? (
              <Tooltip title={period.tooltip}>
                <span>{period.display}</span>
              </Tooltip>
            ) : (
              <span>{period.display}</span>
            )}
            <Typography
              variant="caption"
              color="text.secondary"
              data-testid={`scheduled-task-timezone-${task.id}`}
            >
              {firesIn}
            </Typography>
          </Stack>
        </TableCell>
        <TableCell>
          <RunTime value={task.start_time} />
        </TableCell>
        <TableCell>
          <Stack spacing={0.5} alignItems="flex-start">
            <LastRunStatus
              status={task.last_run_status}
              lastRunAt={task.last_run_at}
              onOpenRun={
                onOpenLastRun
                  ? () => onOpenLastRun(task.task, task.last_run_at)
                  : undefined
              }
            />
            {task.last_run_at && (
              <Typography variant="body2" color="text.secondary">
                <RunTime value={task.last_run_at} />
              </Typography>
            )}
          </Stack>
        </TableCell>
        <TableCell>
          <RunTime value={task.next_run_at} />
        </TableCell>
        <TableCell>{task.total_run_count}</TableCell>
        {showChain && (
          <TableCell>
            {chainNames.length > 0 ? (
              <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                {chainNames.join(' → ')}
              </Typography>
            ) : (
              '—'
            )}
          </TableCell>
        )}
        <TableCell>
          {readOnly ? (
            <Typography variant="body2">
              {task.enabled ? 'Enabled' : 'Disabled'}
            </Typography>
          ) : (
            <Switch
              checked={task.enabled}
              disabled={toggling}
              onChange={(_, checked) => onToggleEnabled(task, checked)}
              slotProps={{
                input: {
                  'aria-label': `Enable ${task.task}`,
                },
              }}
            />
          )}
        </TableCell>
        {readOnly ? null : (
          <TableCell>
            <Stack direction="row" spacing={0.5}>
              <Tooltip title="Edit">
                <IconButton
                  size="small"
                  onClick={onStartEdit}
                  aria-label={`Edit ${task.task}`}
                  data-testid={`scheduled-task-edit-${task.id}`}
                >
                  <EditOutlinedIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Tooltip title="Delete">
                <IconButton
                  size="small"
                  onClick={() => setConfirmOpen(true)}
                  aria-label={`Delete ${task.task}`}
                  data-testid={`scheduled-task-delete-${task.id}`}
                >
                  <DeleteOutlineIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Stack>
          </TableCell>
        )}
      </TableRow>

      <Dialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        aria-labelledby={`scheduled-task-delete-title-${task.id}`}
      >
        <DialogTitle id={`scheduled-task-delete-title-${task.id}`}>
          Delete periodic {itemName}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {`Delete the periodic ${itemName} for "${task.task}" (${period.display})?`}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmOpen(false)}>Cancel</Button>
          <Button
            onClick={() => {
              setConfirmOpen(false);
              onDelete(task);
            }}
            variant="contained"
            autoFocus
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
