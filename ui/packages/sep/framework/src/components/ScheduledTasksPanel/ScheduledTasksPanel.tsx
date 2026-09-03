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

import { useMemo, useState } from 'react';
import AddIcon from '@mui/icons-material/Add';
import ScheduleIcon from '@mui/icons-material/Schedule';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { ScheduledTaskForm } from './ScheduledTaskForm';
import { ScheduledTaskRow } from './ScheduledTaskRow';
import {
  useCreateScheduledTask,
  useDeleteScheduledTask,
  useScheduledTasksForPlugin,
  useUpdateScheduledTask,
  type PeriodicTaskCreate,
  type PeriodicTaskResponse,
  type PeriodicTaskUpdate,
} from './hooks';

interface ScheduledTasksPanelProps {
  pluginName: string;
  /** Disable list polling. Used by stories/tests. */
  disablePolling?: boolean;
}

const COLUMN_HEADERS = [
  'Task',
  'Period',
  'Start Time',
  'Last Run',
  'Next Run',
  'Runs',
  'Chain',
  'Enabled',
  'Actions',
];

export function ScheduledTasksPanel({
  pluginName,
  disablePolling = false,
}: ScheduledTasksPanelProps) {
  const { periodicTasks, pluginTasks, isLoading, isError, error } =
    useScheduledTasksForPlugin(pluginName, { disablePolling });

  const createMut = useCreateScheduledTask();
  const updateMut = useUpdateScheduledTask();
  const deleteMut = useDeleteScheduledTask();

  const [creating, setCreating] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [formError, setFormError] = useState<string | undefined>(undefined);
  const [actionError, setActionError] = useState<string | undefined>(undefined);

  const availableTasks = useMemo(
    () => pluginTasks.map((t) => ({ name: t.name })),
    [pluginTasks]
  );

  const handleToggleEnabled = async (
    task: PeriodicTaskResponse,
    nextEnabled: boolean
  ) => {
    // PeriodicTaskUpdate requires `kwargs` and `description`, but
    // PeriodicTaskResponse declares only `description`. Preserve `kwargs` when
    // the response happens to carry it so a plain enable/disable toggle does
    // not silently wipe a task's arguments; '{}' stays the last-resort
    // fallback. Tracked upstream as a backend schema gap.
    const rawKwargs = (task as { kwargs?: unknown }).kwargs;
    // The wire shape is unverified either way, so accept both: a JSON string
    // passes through, a decoded object is re-serialised. Anything else (or a
    // blank value) falls back to '{}' — the only case that still loses data.
    let preservedKwargs = '{}';
    if (typeof rawKwargs === 'string' && rawKwargs.trim() !== '') {
      preservedKwargs = rawKwargs;
    } else if (rawKwargs !== null && typeof rawKwargs === 'object') {
      preservedKwargs = JSON.stringify(rawKwargs);
    }
    const body: PeriodicTaskUpdate = {
      name: task.name,
      task: task.task,
      enabled: nextEnabled,
      description: task.description,
      kwargs: preservedKwargs,
      start_time: task.start_time,
      interval: task.interval ?? null,
      crontab: task.crontab ?? null,
      execute_request: task.execute_request ?? null,
    };
    setActionError(undefined);
    try {
      await updateMut.mutateAsync({ id: task.id, body });
    } catch (e) {
      setActionError(
        e instanceof Error ? e.message : 'Failed to toggle scheduled task'
      );
    }
  };

  const handleDelete = async (task: PeriodicTaskResponse) => {
    setActionError(undefined);
    try {
      await deleteMut.mutateAsync(task.id);
    } catch (e) {
      setActionError(
        e instanceof Error ? e.message : 'Failed to delete scheduled task'
      );
    }
  };

  const handleCreate = async (
    body: PeriodicTaskCreate | PeriodicTaskUpdate,
    taskName: string
  ) => {
    setFormError(undefined);
    try {
      await createMut.mutateAsync({
        taskName,
        body: body as PeriodicTaskCreate,
      });
      setCreating(false);
    } catch (e) {
      setFormError(
        e instanceof Error ? e.message : 'Failed to create scheduled task'
      );
    }
  };

  const handleEditSubmit = (task: PeriodicTaskResponse) => {
    return async (body: PeriodicTaskCreate | PeriodicTaskUpdate) => {
      setFormError(undefined);
      try {
        await updateMut.mutateAsync({
          id: task.id,
          body: body as PeriodicTaskUpdate,
        });
        setEditingId(null);
      } catch (e) {
        setFormError(
          e instanceof Error ? e.message : 'Failed to update scheduled task'
        );
      }
    };
  };

  const startCreate = () => {
    setEditingId(null);
    setFormError(undefined);
    setCreating(true);
  };

  const startEdit = (id: number) => {
    setCreating(false);
    setFormError(undefined);
    setEditingId(id);
  };

  const headerRow = (
    <TableHead>
      <TableRow>
        {COLUMN_HEADERS.map((h) => (
          <TableCell key={h}>{h}</TableCell>
        ))}
      </TableRow>
    </TableHead>
  );

  if (isLoading) {
    return (
      <Paper variant="outlined" sx={{ p: 3, textAlign: 'center' }}>
        <CircularProgress size={24} />
      </Paper>
    );
  }

  if (isError) {
    return (
      <Alert severity="error">
        Failed to load scheduled tasks{error ? `: ${error.message}` : ''}
      </Alert>
    );
  }

  const isEmpty = periodicTasks.length === 0 && !creating;

  return (
    <Paper variant="outlined" data-testid="scheduled-tasks-panel">
      <Box sx={{ p: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
        <ScheduleIcon fontSize="small" />
        <Typography variant="h6">Scheduled Tasks</Typography>
      </Box>

      {actionError && (
        <Alert
          severity="error"
          onClose={() => setActionError(undefined)}
          sx={{ mx: 2, mb: 1 }}
          data-testid="scheduled-tasks-action-error"
        >
          {actionError}
        </Alert>
      )}

      {isEmpty ? (
        <Box sx={{ p: 3, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            No scheduled tasks for {pluginName}.
          </Typography>
        </Box>
      ) : (
        <TableContainer>
          <Table size="small">
            {headerRow}
            <TableBody>
              {periodicTasks.map((task) => (
                <ScheduledTaskRow
                  key={task.id}
                  task={task}
                  availableTasks={availableTasks}
                  isEditing={editingId === task.id}
                  onStartEdit={() => startEdit(task.id)}
                  onCancelEdit={() => setEditingId(null)}
                  onToggleEnabled={handleToggleEnabled}
                  onSubmitEdit={handleEditSubmit(task)}
                  onDelete={handleDelete}
                  submitting={updateMut.isPending}
                  toggling={updateMut.isPending}
                  errorMessage={editingId === task.id ? formError : undefined}
                />
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {creating && (
        <Box sx={{ borderTop: 1, borderColor: 'divider' }}>
          <ScheduledTaskForm
            mode="create"
            availableTasks={availableTasks}
            defaultTaskName={availableTasks[0]?.name}
            onCancel={() => setCreating(false)}
            onSubmit={handleCreate}
            submitting={createMut.isPending}
            errorMessage={formError}
          />
        </Box>
      )}

      {!creating && (
        <Stack
          direction="row"
          justifyContent="flex-end"
          sx={{ p: 1, borderTop: 1, borderColor: 'divider' }}
        >
          <Button
            startIcon={<AddIcon />}
            onClick={startCreate}
            disabled={availableTasks.length === 0}
            data-testid="scheduled-tasks-add"
          >
            Add new
          </Button>
        </Stack>
      )}
    </Paper>
  );
}
