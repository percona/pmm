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
import { describePeriod, formatAbsoluteTime } from './periods';
import type { AvailableTask } from '../ChainBuilder';
import type {
  PeriodicTaskCreate,
  PeriodicTaskResponse,
  PeriodicTaskUpdate,
} from './hooks';

const COLUMN_COUNT = 9;

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
}: ScheduledTaskRowProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const period = describePeriod(task);
  const chainNames = task.execute_request?.chain_task_names ?? [];

  if (isEditing) {
    return (
      <TableRow>
        <TableCell colSpan={COLUMN_COUNT} sx={{ p: 0 }}>
          <ScheduledTaskForm
            mode="edit"
            initialValue={task}
            availableTasks={availableTasks}
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
          {period.tooltip ? (
            <Tooltip title={period.tooltip}>
              <span>{period.display}</span>
            </Tooltip>
          ) : (
            period.display
          )}
        </TableCell>
        <TableCell>{formatAbsoluteTime(task.start_time)}</TableCell>
        <TableCell>
          <Stack spacing={0.5} alignItems="flex-start">
            <LastRunStatus
              status={task.last_run_status}
              lastRunAt={task.last_run_at}
            />
            {task.last_run_at && (
              <Typography variant="body2" color="text.secondary">
                {formatAbsoluteTime(task.last_run_at)}
              </Typography>
            )}
          </Stack>
        </TableCell>
        <TableCell>{formatAbsoluteTime(task.next_run_at)}</TableCell>
        <TableCell>{task.total_run_count}</TableCell>
        <TableCell>
          {chainNames.length > 0 ? (
            <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
              {chainNames.join(' → ')}
            </Typography>
          ) : (
            '—'
          )}
        </TableCell>
        <TableCell>
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
        </TableCell>
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
      </TableRow>

      <Dialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        aria-labelledby={`scheduled-task-delete-title-${task.id}`}
      >
        <DialogTitle id={`scheduled-task-delete-title-${task.id}`}>
          Delete periodic task
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {`Delete the periodic task for "${task.task}" (${period.display})?`}
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
