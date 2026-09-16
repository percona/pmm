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

import { useCallback, useMemo, useState } from 'react';
import AddIcon from '@mui/icons-material/Add';
import ScheduleIcon from '@mui/icons-material/Schedule';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import Stack from '@mui/material/Stack';
import type { TableRowProps } from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { MaterialReactTable } from 'material-react-table';
import { useAuth } from '@sep/api';
import { capitalize } from '@sep/shared';
import { scheduleColumns } from './columns';
import { describePeriod } from './periods';
import { browserTimezone } from './timezones';
import { sepTableProps } from '../SepTable';
import { TaskRunDetailDrawer } from '../TaskRunDetailDrawer';
import {
  useDeleteScheduledTask,
  useScheduledTasksForPlugin,
  useUpdateScheduledTask,
  type PeriodicTaskResponse,
  type PeriodicTaskUpdate,
} from './hooks';

interface ScheduledTasksPanelProps {
  pluginName: string;
  /**
   * What to call the app in the empty state. `pluginName` is a registry key —
   * a nested app's is scoped (`mysql_backups/restore`) — so it is the wrong
   * thing to show a reader. Falls back to `pluginName` for a caller that has no
   * schema to take a display name from.
   */
  displayName?: string;
  /** Disable list polling. Used by stories/tests. */
  disablePolling?: boolean;
  /** Mid-sentence singular noun for one record (e.g. `backup`). */
  itemName?: string;
  /** Mid-sentence plural noun (e.g. `backups`). */
  itemNamePlural?: string;
  /**
   * Open the create form. Omit to hide the header's create action — for a host
   * that has nowhere to send the reader.
   *
   * A callback rather than internal state: creating and editing a schedule is
   * a page of its own now, the same pattern a plugin task already used
   * (PMM-15456), and this panel does not own the app's routes.
   */
  onCreate?: () => void;
  /** Open the edit form for one schedule. Omit to hide the row's edit action. */
  onEdit?: (task: PeriodicTaskResponse) => void;
}

export function ScheduledTasksPanel({
  pluginName,
  displayName,
  disablePolling = false,
  itemName = 'task',
  itemNamePlural = 'tasks',
  onCreate,
  onEdit,
}: ScheduledTasksPanelProps) {
  const { canMutate } = useAuth();
  const itemLabel = capitalize(itemName);
  const itemPluralLabel = capitalize(itemNamePlural);
  const { periodicTasks, pluginTasks, isLoading, isError, error } =
    useScheduledTasksForPlugin(pluginName, { disablePolling });

  const updateMut = useUpdateScheduledTask();
  const deleteMut = useDeleteScheduledTask();

  const [actionError, setActionError] = useState<string | undefined>(undefined);
  const [pendingDelete, setPendingDelete] =
    useState<PeriodicTaskResponse | null>(null);
  // The periodic-task API reports a last-run status and time but no task
  // history id, so the cell identifies its run by name plus that timestamp.
  const [openedRun, setOpenedRun] = useState<{
    taskName: string;
    lastRunAt: string | null;
  } | null>(null);

  // Show the Chain column only once something is actually chained. Derived from
  // the rows rather than a capability flag: the column's job is to display a
  // chain, so the presence of one is the honest condition.
  const showChain = useMemo(
    () =>
      periodicTasks.some(
        (t) => (t.execute_request?.chain_task_names?.length ?? 0) > 0
      ),
    [periodicTasks]
  );

  // Every absolute time in this table goes through `toLocaleString()`, so it is
  // rendered in the reader's zone — not the zone the schedule fires in, which
  // each row states for itself. Naming it here is what keeps the two readable
  // side by side (PMM-15454).
  const displayZone = browserTimezone();

  const handleToggleEnabled = useCallback(
    async (task: PeriodicTaskResponse, nextEnabled: boolean) => {
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
          e instanceof Error
            ? e.message
            : `Failed to toggle scheduled ${itemName}`
        );
      }
    },
    [itemName, updateMut]
  );

  const handleDelete = async (task: PeriodicTaskResponse) => {
    setActionError(undefined);
    try {
      await deleteMut.mutateAsync(task.id);
    } catch (e) {
      setActionError(
        e instanceof Error
          ? e.message
          : `Failed to delete scheduled ${itemName}`
      );
    }
  };

  const openLastRun = useCallback(
    (taskName: string, lastRunAt: string | null) =>
      setOpenedRun({ taskName, lastRunAt }),
    []
  );

  const columns = useMemo(
    () =>
      scheduleColumns({
        showChain,
        canMutate,
        toggling: updateMut.isPending,
        onToggleEnabled: handleToggleEnabled,
        onEdit,
        onDelete: setPendingDelete,
        onOpenLastRun: openLastRun,
        itemLabel,
      }),
    [
      showChain,
      canMutate,
      updateMut.isPending,
      handleToggleEnabled,
      onEdit,
      openLastRun,
      itemLabel,
    ]
  );

  const canCreate = canMutate && onCreate !== undefined;
  const hasCreatableTask = pluginTasks.length > 0;

  if (isError) {
    return (
      <Alert severity="error">
        Failed to load scheduled {itemNamePlural}
        {error ? `: ${error.message}` : ''}
      </Alert>
    );
  }

  const isEmpty = !isLoading && periodicTasks.length === 0;

  return (
    <Box data-testid="scheduled-tasks-panel">
      {/*
        Title on the left, actions on the right — the header row a SEP list
        has (see PluginListPage). "Add new" used to sit under the table in a
        footer strip, which is where nothing else in the app puts its primary
        action (PMM-15456).
      */}
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        gap={2}
        sx={{ mb: 2 }}
      >
        <Stack direction="row" alignItems="center" gap={1}>
          <ScheduleIcon fontSize="small" />
          <Typography variant="h6">Scheduled {itemPluralLabel}</Typography>
        </Stack>
        <Stack direction="row" alignItems="center" gap={2}>
          {!isEmpty && (
            <Typography
              variant="caption"
              color="text.secondary"
              data-testid="scheduled-tasks-display-timezone"
            >
              Times shown in {displayZone}
            </Typography>
          )}
          {canCreate && (
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={onCreate}
              disabled={!hasCreatableTask}
              data-testid="scheduled-tasks-add"
            >
              New schedule
            </Button>
          )}
        </Stack>
      </Stack>

      {actionError && (
        <Alert
          severity="error"
          onClose={() => setActionError(undefined)}
          sx={{ mb: 2 }}
          data-testid="scheduled-tasks-action-error"
        >
          {actionError}
        </Alert>
      )}

      <MaterialReactTable
        {...sepTableProps<PeriodicTaskResponse>()}
        columns={columns}
        data={periodicTasks}
        state={{ isLoading }}
        getRowId={(row) => String(row.id)}
        enableTopToolbar={false}
        enableSorting
        enablePagination
        initialState={{
          density: 'compact',
          pagination: { pageIndex: 0, pageSize: 10 },
        }}
        muiTableBodyRowProps={({ row }) =>
          ({
            'data-testid': `scheduled-task-row-${row.original.id}`,
          }) as TableRowProps
        }
        renderEmptyRowsFallback={() => (
          <Box sx={{ p: 3, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">
              No scheduled {itemNamePlural} for {displayName ?? pluginName}.
            </Typography>
          </Box>
        )}
      />

      <Dialog
        open={pendingDelete !== null}
        onClose={() => setPendingDelete(null)}
        aria-labelledby="scheduled-task-delete-title"
      >
        <DialogTitle id="scheduled-task-delete-title">
          Delete periodic {itemName}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {pendingDelete
              ? `Delete the periodic ${itemName} for "${pendingDelete.task}" (${describePeriod(pendingDelete).display})?`
              : ''}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPendingDelete(null)}>Cancel</Button>
          <Button
            onClick={() => {
              const task = pendingDelete;
              setPendingDelete(null);
              if (task) {
                void handleDelete(task);
              }
            }}
            variant="contained"
            autoFocus
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {openedRun !== null && (
        <TaskRunDetailDrawer
          open
          onClose={() => setOpenedRun(null)}
          taskNames={openedRun.taskName}
          at={openedRun.lastRunAt}
          taskLabel={openedRun.taskName}
          itemName={itemName}
        />
      )}
    </Box>
  );
}
