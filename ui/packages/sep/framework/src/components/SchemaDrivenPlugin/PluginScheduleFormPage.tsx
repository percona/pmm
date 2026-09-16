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
import { useNavigate, useParams } from 'react-router-dom';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { useAuth } from '@sep/api';
import { ReadOnlyNotice } from '../ReadOnlyNotice';
import { ScheduledTaskForm } from '../ScheduledTasksPanel/ScheduledTaskForm';
import {
  useCreateScheduledTask,
  useScheduledTasksForPlugin,
  useUpdateScheduledTask,
  type PeriodicTaskCreate,
  type PeriodicTaskUpdate,
} from '../ScheduledTasksPanel/hooks';

export interface PluginScheduleFormPageProps {
  pluginName: string;
  /** `create` takes its default task from the plugin; `edit` reads `:id`. */
  mode: 'create' | 'edit';
  /** Mid-sentence singular noun for one record (e.g. `backup`). */
  itemName?: string;
  /** Mid-sentence plural noun (e.g. `backups`). */
  itemNamePlural?: string;
}

/**
 * Create or edit one schedule, as a page.
 *
 * A schedule used to be edited by expanding the table row it lived in, while a
 * plugin task was edited on a page of its own — two answers to the same
 * question, in one app. This is the page (PMM-15456), and it carries the same
 * chrome as `PluginCreatePage` / `PluginTaskEditPage`: back arrow, heading, and
 * the read-only guard for a session that arrived by URL.
 */
export function PluginScheduleFormPage({
  pluginName,
  mode,
  itemName = 'task',
  itemNamePlural = 'tasks',
}: PluginScheduleFormPageProps) {
  const navigate = useNavigate();
  const { canMutate } = useAuth();
  const { id } = useParams<{ id: string }>();
  const { periodicTasks, pluginTasks, isLoading } =
    useScheduledTasksForPlugin(pluginName);

  const createMut = useCreateScheduledTask();
  const updateMut = useUpdateScheduledTask();
  const [formError, setFormError] = useState<string | undefined>(undefined);

  const availableTasks = useMemo(
    () => pluginTasks.map((t) => ({ name: t.name })),
    [pluginTasks]
  );
  const initialValue = useMemo(
    () => periodicTasks.find((t) => String(t.id) === id),
    [periodicTasks, id]
  );

  // The schedules list is one route up from both `new` and `:id/edit`, so the
  // two forms leave by the same relative path.
  const backToList = () =>
    navigate(mode === 'create' ? '..' : '../..', { relative: 'path' });

  const title = mode === 'create' ? 'New schedule' : 'Edit schedule';

  const chrome = (children: React.ReactNode) => (
    <Box>
      <Stack direction="row" alignItems="center" gap={1} sx={{ mb: 3 }}>
        <IconButton onClick={backToList} aria-label="Back to schedules">
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h4">{title}</Typography>
      </Stack>
      {children}
    </Box>
  );

  if (!canMutate) {
    return chrome(
      <ReadOnlyNotice
        action="change schedules"
        testId="plugin-schedule-read-only"
      />
    );
  }

  if (isLoading) {
    return chrome(
      <Stack alignItems="center" py={4}>
        <CircularProgress size={24} data-testid="plugin-schedule-loading" />
      </Stack>
    );
  }

  if (mode === 'edit' && !initialValue) {
    return chrome(
      <Typography variant="body2" color="text.secondary">
        That schedule no longer exists.
      </Typography>
    );
  }

  const handleSubmit = async (
    body: PeriodicTaskCreate | PeriodicTaskUpdate,
    taskName: string
  ) => {
    setFormError(undefined);
    try {
      if (mode === 'create') {
        await createMut.mutateAsync({
          taskName,
          body: body as PeriodicTaskCreate,
        });
      } else {
        await updateMut.mutateAsync({
          id: initialValue!.id,
          body: body as PeriodicTaskUpdate,
        });
      }
      backToList();
    } catch (e) {
      setFormError(
        e instanceof Error
          ? e.message
          : `Failed to ${mode === 'create' ? 'create' : 'update'} scheduled ${itemName}`
      );
    }
  };

  return chrome(
    <ScheduledTaskForm
      mode={mode}
      initialValue={initialValue}
      availableTasks={availableTasks}
      defaultTaskName={availableTasks[0]?.name}
      itemName={itemName}
      itemNamePlural={itemNamePlural}
      onCancel={backToList}
      onSubmit={handleSubmit}
      submitting={createMut.isPending || updateMut.isPending}
      errorMessage={formError}
    />
  );
}
