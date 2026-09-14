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

import { useEffect, useMemo } from 'react';
import Alert from '@mui/material/Alert';
import Autocomplete from '@mui/material/Autocomplete';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import FormControlLabel from '@mui/material/FormControlLabel';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { Controller, useForm, type SubmitHandler } from 'react-hook-form';
import cronstrue from 'cronstrue';
import { capitalize } from '@sep/shared';
import {
  ChainBuilder,
  type AvailableTask,
  type ChainValue,
} from '../ChainBuilder';
import {
  INTERVAL_TIMEZONE,
  TIMEZONES,
  defaultPickerTimezone,
  utcInputToIso,
  utcIsoToUtcInput,
} from './timezones';
import type {
  CrontabSchedule,
  IntervalSchedule,
  PeriodicTaskCreate,
  PeriodicTaskResponse,
  PeriodicTaskUpdate,
} from './hooks';

export type IntervalUnit = 'days' | 'hours' | 'minutes';

export interface ScheduledTaskFormValues {
  task: string;
  scheduleMode: 'interval' | 'cron';
  intervalEvery: number;
  intervalPeriod: IntervalUnit;
  cronExpression: string;
  cronTimezone: string;
  startTime: string;
  enabled: boolean;
  chain: ChainValue;
}

export interface ScheduledTaskFormProps {
  mode: 'create' | 'edit';
  initialValue?: PeriodicTaskResponse;
  availableTasks: AvailableTask[];
  defaultTaskName?: string;
  onCancel: () => void;
  onSubmit: (
    body: PeriodicTaskCreate | PeriodicTaskUpdate,
    taskName: string
  ) => Promise<void>;
  submitting?: boolean;
  errorMessage?: string;
  /** Mid-sentence singular noun for one record (e.g. `backup`). */
  itemName?: string;
  /** Mid-sentence plural noun (e.g. `backups`). */
  itemNamePlural?: string;
}

const CRON_PATTERN = /^\S+(?:\s+\S+){4}$/;

/**
 * The `start_time` to send.
 *
 * The field carries minutes, so round-tripping a stored value through it drops
 * any seconds that value had. Send the stored instant back verbatim while the
 * field still shows it unchanged, so saving an edit to an unrelated field —
 * the interval, the enable toggle — cannot quietly move a schedule's first fire
 * by up to a minute (PMM-15454).
 */
function startTimeToSubmit(
  fieldValue: string,
  stored: string | null | undefined
): string | null {
  if (stored && fieldValue === utcIsoToUtcInput(stored)) {
    return stored;
  }
  return utcInputToIso(fieldValue);
}

function cronToExpression(c: CrontabSchedule): string {
  return `${c.minute} ${c.hour} ${c.day_of_month} ${c.month_of_year} ${c.day_of_week}`;
}

function expressionToCron(
  expr: string,
  timezone: string
): CrontabSchedule | null {
  const parts = expr.trim().split(/\s+/);
  if (parts.length !== 5) {
    return null;
  }
  const [minute, hour, day_of_month, month_of_year, day_of_week] = parts;
  return { minute, hour, day_of_month, month_of_year, day_of_week, timezone };
}

function humanize(expr: string): { text: string; valid: boolean } {
  if (!CRON_PATTERN.test(expr.trim())) {
    return { text: 'Invalid cron expression', valid: false };
  }
  try {
    const text = cronstrue.toString(expr.trim());
    return { text: text.charAt(0).toLowerCase() + text.slice(1), valid: true };
  } catch {
    return { text: 'Invalid cron expression', valid: false };
  }
}

function buildDefaults(
  initial: PeriodicTaskResponse | undefined,
  defaultTaskName: string | undefined
): ScheduledTaskFormValues {
  if (initial) {
    return {
      task: initial.task,
      scheduleMode: initial.crontab ? 'cron' : 'interval',
      intervalEvery: initial.interval?.every ?? 1,
      intervalPeriod: (initial.interval?.period as IntervalUnit) ?? 'hours',
      cronExpression: initial.crontab ? cronToExpression(initial.crontab) : '',
      cronTimezone: initial.crontab?.timezone ?? defaultPickerTimezone(),
      startTime: initial.start_time ? utcIsoToUtcInput(initial.start_time) : '',
      enabled: initial.enabled,
      chain: {
        chain_task_names: initial.execute_request?.chain_task_names ?? [],
        chain_on_failure: initial.execute_request?.chain_on_failure ?? false,
      },
    };
  }
  return {
    task: defaultTaskName ?? '',
    scheduleMode: 'interval',
    intervalEvery: 1,
    intervalPeriod: 'hours',
    cronExpression: '',
    cronTimezone: defaultPickerTimezone(),
    startTime: '',
    enabled: true,
    chain: { chain_task_names: [], chain_on_failure: false },
  };
}

export function ScheduledTaskForm({
  mode,
  initialValue,
  availableTasks,
  defaultTaskName,
  onCancel,
  onSubmit,
  submitting = false,
  errorMessage,
  itemName = 'task',
  itemNamePlural = 'tasks',
}: ScheduledTaskFormProps) {
  const itemLabel = capitalize(itemName);
  const defaults = useMemo(
    () => buildDefaults(initialValue, defaultTaskName),
    [initialValue, defaultTaskName]
  );

  const {
    control,
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<ScheduledTaskFormValues>({ defaultValues: defaults });

  const scheduleMode = watch('scheduleMode');
  const cronExpression = watch('cronExpression');
  const cronTimezone = watch('cronTimezone');
  const timezoneOptions = useMemo(
    () =>
      cronTimezone && !TIMEZONES.includes(cronTimezone)
        ? [cronTimezone, ...TIMEZONES]
        : TIMEZONES,
    [cronTimezone]
  );
  const taskName = watch('task');
  const chain = watch('chain');

  // When the user changes the task in create mode, prune the chain so it
  // never contains the now-current task name (would form a self-cycle).
  useEffect(() => {
    if (!taskName) {
      return;
    }
    if (chain.chain_task_names.includes(taskName)) {
      setValue(
        'chain',
        {
          ...chain,
          chain_task_names: chain.chain_task_names.filter(
            (n) => n !== taskName
          ),
        },
        { shouldDirty: true }
      );
    }
  }, [taskName, chain, setValue]);

  const cronPreview = useMemo(() => {
    if (scheduleMode !== 'cron' || !cronExpression) {
      return null;
    }
    return humanize(cronExpression);
  }, [scheduleMode, cronExpression]);

  const toggleMode = () => {
    setValue(
      'scheduleMode',
      scheduleMode === 'interval' ? 'cron' : 'interval',
      {
        shouldDirty: true,
      }
    );
  };

  const submit: SubmitHandler<ScheduledTaskFormValues> = async (values) => {
    const isCron = values.scheduleMode === 'cron';
    let crontab: CrontabSchedule | null = null;
    if (isCron) {
      const preview = humanize(values.cronExpression);
      if (!preview.valid) {
        return;
      }
      crontab = expressionToCron(values.cronExpression, values.cronTimezone);
      if (!crontab) {
        return;
      }
    }

    const everyNum = Number(values.intervalEvery);
    if (!isCron && (!Number.isFinite(everyNum) || everyNum < 1)) {
      return;
    }
    const interval: IntervalSchedule | null = isCron
      ? null
      : { every: everyNum, period: values.intervalPeriod };

    // The start-time field is labelled UTC and carries UTC wall clock, so it
    // reads back as UTC. Cron mode asks no start time; that half of the finding
    // is parked for a design pass (PMM-15454).
    const start_time = isCron
      ? null
      : startTimeToSubmit(values.startTime, initialValue?.start_time);

    const hasChain = values.chain.chain_task_names.length > 0;
    const execute_request = hasChain
      ? {
          meta: {} as Record<string, never>,
          chain_task_names: values.chain.chain_task_names,
          chain_on_failure: values.chain.chain_on_failure,
        }
      : null;

    const body: PeriodicTaskCreate | PeriodicTaskUpdate = {
      name: initialValue?.name ?? '',
      task: values.task,
      enabled: values.enabled,
      description: initialValue?.description ?? '',
      kwargs: '{}',
      start_time,
      interval,
      crontab,
      execute_request,
    };

    await onSubmit(body, values.task);
  };

  const taskField =
    mode === 'create' ? (
      <TextField
        select
        size="small"
        label={itemLabel}
        required
        slotProps={{ htmlInput: { 'data-testid': 'sched-form-task' } }}
        {...register('task', { required: true })}
        defaultValue={defaults.task}
        error={!!errors.task}
        sx={{ minWidth: 180 }}
      >
        {availableTasks.map((t) => (
          <MenuItem key={t.name} value={t.name}>
            {t.name}
          </MenuItem>
        ))}
      </TextField>
    ) : (
      <Typography variant="body2" sx={{ alignSelf: 'center' }}>
        {taskName}
      </Typography>
    );

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(submit)}
      data-testid="scheduled-task-form"
      sx={{ p: 2, bgcolor: 'action.hover' }}
    >
      {errorMessage && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {errorMessage}
        </Alert>
      )}

      <Stack
        direction="row"
        spacing={2}
        flexWrap="wrap"
        alignItems="flex-start"
        sx={{ mb: 2 }}
      >
        {taskField}

        {scheduleMode === 'interval' ? (
          <Stack direction="row" spacing={1} alignItems="center">
            <TextField
              type="number"
              size="small"
              label="Every"
              required
              slotProps={{
                htmlInput: {
                  min: 1,
                  'data-testid': 'sched-form-interval-every',
                },
              }}
              {...register('intervalEvery', {
                required: true,
                valueAsNumber: true,
                validate: (v) => Number.isFinite(v) && (v as number) >= 1,
              })}
              error={!!errors.intervalEvery}
              sx={{ width: 100 }}
            />
            <TextField
              select
              size="small"
              label="Period"
              {...register('intervalPeriod')}
              defaultValue={defaults.intervalPeriod}
              sx={{ width: 120 }}
            >
              <MenuItem value="days">days</MenuItem>
              <MenuItem value="hours">hours</MenuItem>
              <MenuItem value="minutes">minutes</MenuItem>
            </TextField>
          </Stack>
        ) : (
          <Stack spacing={0.5} sx={{ minWidth: 320 }}>
            <Stack direction="row" spacing={1}>
              <TextField
                size="small"
                label="Cron expression"
                placeholder="*/5 * * * *"
                required
                slotProps={{
                  htmlInput: {
                    pattern: '^\\S+(?:\\s+\\S+){4}$',
                    'data-testid': 'sched-form-cron',
                  },
                }}
                {...register('cronExpression', {
                  required: true,
                  validate: (v) =>
                    humanize(v).valid || 'Invalid cron expression',
                })}
                error={!!errors.cronExpression}
                sx={{ flex: 1 }}
              />
              <Controller
                control={control}
                name="cronTimezone"
                render={({ field }) => (
                  <Autocomplete
                    size="small"
                    options={timezoneOptions}
                    value={field.value}
                    onChange={(_, v) => field.onChange(v ?? 'UTC')}
                    sx={{ width: 220 }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Timezone"
                        slotProps={{
                          htmlInput: {
                            ...params.inputProps,
                            'data-testid': 'sched-form-timezone',
                          },
                        }}
                      />
                    )}
                  />
                )}
              />
            </Stack>
            {cronPreview && (
              <Typography
                variant="caption"
                color={cronPreview.valid ? 'text.secondary' : 'error'}
                data-testid="sched-form-cron-preview"
              >
                {cronPreview.text}
              </Typography>
            )}
          </Stack>
        )}

        {scheduleMode === 'interval' && (
          <TextField
            type="datetime-local"
            size="small"
            label={`Start time (${INTERVAL_TIMEZONE})`}
            slotProps={{
              inputLabel: { shrink: true },
              htmlInput: { 'data-testid': 'sched-form-start-time' },
            }}
            {...register('startTime')}
            sx={{ width: 220 }}
          />
        )}

        <Controller
          control={control}
          name="enabled"
          render={({ field }) => (
            <FormControlLabel
              sx={{ alignSelf: 'center' }}
              control={
                <Switch
                  checked={field.value}
                  onChange={(_, c) => field.onChange(c)}
                  inputProps={
                    {
                      'data-testid': 'sched-form-enabled',
                    } as React.InputHTMLAttributes<HTMLInputElement>
                  }
                />
              }
              label="Enabled"
            />
          )}
        />
      </Stack>

      <Box sx={{ mb: 1 }}>
        <Typography
          variant="caption"
          color="text.secondary"
          data-testid="sched-form-timezone-notice"
        >
          {scheduleMode === 'cron'
            ? `Runs in ${cronTimezone}.`
            : `Runs in ${INTERVAL_TIMEZONE} — an interval schedule has no timezone of its own.`}
        </Typography>
      </Box>

      <Box sx={{ mb: 1 }}>
        <Link
          component="button"
          type="button"
          variant="caption"
          onClick={toggleMode}
          data-testid="sched-form-toggle-mode"
        >
          {scheduleMode === 'interval'
            ? 'change to cron mode'
            : 'change to interval mode'}
        </Link>
      </Box>

      <Controller
        control={control}
        name="chain"
        render={({ field }) => (
          <ChainBuilder
            availableTasks={availableTasks}
            currentTaskName={taskName}
            value={field.value}
            onChange={field.onChange}
            itemName={itemName}
            itemNamePlural={itemNamePlural}
            sx={{ mb: 2 }}
          />
        )}
      />

      <Stack direction="row" spacing={1} justifyContent="flex-end">
        <Button onClick={onCancel} disabled={submitting} type="button">
          Cancel
        </Button>
        <Button type="submit" variant="contained" disabled={submitting}>
          {mode === 'create' ? 'Create' : 'Save'}
        </Button>
      </Stack>
    </Box>
  );
}
