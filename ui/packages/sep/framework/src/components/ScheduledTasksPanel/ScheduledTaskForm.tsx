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

import { useEffect, useMemo, useState } from 'react';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import {
  AutoCompleteInput,
  RadioGroup,
  SelectInput,
  SwitchInput,
  TextInput,
} from '@percona/peak-ui';
import { Controller, useForm, type SubmitHandler } from 'react-hook-form';
import cronstrue from 'cronstrue';
import { capitalize } from '@sep/shared';
import { DateTimeInput } from '../DateTimeInput';
import { FORM_CONTENT_MAX_WIDTH } from '../../constants';
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
import {
  useSchedulePreview,
  type CrontabSchedule,
  type SchedulePreviewWrite,
  type IntervalSchedule,
  type PeriodicTaskCreate,
  type PeriodicTaskResponse,
  type PeriodicTaskUpdate,
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
 * Hold a value still for `delay` ms.
 *
 * Without it the schedule preview is a network call per keystroke, and a
 * half-typed cron expression is frequently a valid one in its own right — so
 * the requests would not merely be many, they would each describe a schedule
 * the user never asked for.
 */
function useDebounced<T>(value: T, delay = 400): T {
  const [held, setHeld] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setHeld(value), delay);
    return () => clearTimeout(t);
  }, [value, delay]);
  return held;
}

/**
 * The `start_time` to send.
 *
 * The field carries minutes, so round-tripping a stored value through it drops
 * any seconds that value had. Send the stored instant back verbatim while the
 * reader has not touched the field, so saving an edit to an unrelated field —
 * the interval, the enable toggle — cannot quietly move a schedule's first fire
 * (PMM-15454).
 *
 * Keyed on whether the field is dirty rather than on comparing its value to the
 * stored one. The picker renders through a local `Date`, which cannot represent
 * a wall clock inside the reader's own spring-forward gap (see
 * `DateTimeInput/wallClockValue.ts`); a value comparison would read that
 * one-hour display shift as an edit and persist it. Dirtiness answers the
 * question actually being asked — did anyone change this? (PMM-15456)
 */
function startTimeToSubmit(
  fieldValue: string,
  stored: string | null | undefined,
  touched: boolean
): string | null {
  if (stored && !touched) {
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
    handleSubmit,
    watch,
    setValue,
    formState: { dirtyFields },
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

  const intervalEvery = watch('intervalEvery');
  const intervalPeriod = watch('intervalPeriod');
  const startTime = watch('startTime');

  // What the backend would make of the schedule as currently described. Null
  // while the form does not describe a valid one, which keeps the query idle.
  const previewSpec = useMemo<SchedulePreviewWrite | null>(() => {
    if (scheduleMode === 'cron') {
      if (!cronExpression || !humanize(cronExpression).valid) {
        return null;
      }
      const crontab = expressionToCron(cronExpression, cronTimezone);
      return crontab ? { crontab, interval: null, start_time: null } : null;
    }
    const every = Number(intervalEvery);
    if (!Number.isFinite(every) || every < 1) {
      return null;
    }
    return {
      interval: { every, period: intervalPeriod },
      crontab: null,
      start_time: utcInputToIso(startTime),
    };
  }, [
    scheduleMode,
    cronExpression,
    cronTimezone,
    intervalEvery,
    intervalPeriod,
    startTime,
  ]);

  const { data: preview, isError: previewFailed } = useSchedulePreview(
    useDebounced(previewSpec)
  );

  // The backend's own answer beats the form's assumption about which zone is in
  // force; fall back to the assumption until the first preview lands.
  const zoneInForce =
    preview?.timezone ??
    (scheduleMode === 'cron' ? cronTimezone : INTERVAL_TIMEZONE);

  const nextRuns = preview?.next_runs?.slice(0, 3) ?? [];

  // A clock time in the zone the caption above already names — never relative
  // and never the reader's own zone, since a run in "2 hours" or in the
  // reader's zone would silently contradict "Runs in {zoneInForce}" above it.
  const formatNextRun = (value: string): string => {
    const target = new Date(value);
    if (Number.isNaN(target.getTime())) {
      return value;
    }
    return target.toLocaleString(undefined, {
      timeZone: zoneInForce,
      dateStyle: 'medium',
      timeStyle: 'short',
    });
  };

  const cronPreview = useMemo(() => {
    if (scheduleMode !== 'cron' || !cronExpression) {
      return null;
    }
    return humanize(cronExpression);
  }, [scheduleMode, cronExpression]);

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
      : startTimeToSubmit(
          values.startTime,
          initialValue?.start_time,
          Boolean(dirtyFields.startTime)
        );

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
      <SelectInput
        name="task"
        control={control}
        label={itemLabel}
        controllerProps={{ rules: { required: true } }}
        formControlProps={{ fullWidth: true, size: 'small', required: true }}
      >
        {availableTasks.map((t) => (
          <MenuItem key={t.name} value={t.name}>
            {t.name}
          </MenuItem>
        ))}
      </SelectInput>
    ) : (
      <Box>
        <Typography variant="caption" color="text.secondary" display="block">
          {itemLabel}
        </Typography>
        <Typography variant="body1" data-testid="sched-form-task-name">
          {taskName}
        </Typography>
      </Box>
    );

  return (
    <Box
      component="form"
      onSubmit={handleSubmit(submit)}
      data-testid="scheduled-task-form"
      noValidate
      sx={{ maxWidth: FORM_CONTENT_MAX_WIDTH }}
    >
      {errorMessage && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {errorMessage}
        </Alert>
      )}

      <Stack spacing={2.5}>
        {taskField}

        {/*
          A labelled choice rather than the lowercase "change to cron mode"
          link this replaced: the two branches below are alternative answers to
          one question, and a text link neither said which one was in force nor
          looked like anything else on the form (PMM-15456).
        */}
        <RadioGroup
          name="scheduleMode"
          control={control}
          label="Schedule"
          radioGroupFieldProps={{ row: true }}
          options={[
            { label: 'Every so often', value: 'interval' },
            { label: 'On a cron expression', value: 'cron' },
          ]}
        />

        {scheduleMode === 'interval' ? (
          <>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
              <TextInput
                name="intervalEvery"
                control={control}
                label="Every"
                isRequired
                controllerProps={{
                  rules: {
                    required: 'Enter how often this runs',
                    validate: (v) =>
                      (Number.isFinite(Number(v)) && Number(v) >= 1) ||
                      'Must be at least 1',
                  },
                }}
                textFieldProps={{
                  type: 'number',
                  size: 'small',
                  slotProps: {
                    // Setting `slotProps` replaces Peak UI's own, which is
                    // where its `text-input-<field>` test id lives — so the id
                    // has to be restated alongside the spinner bound.
                    htmlInput: {
                      min: 1,
                      'data-testid': 'text-input-interval-every',
                    },
                  },
                  sx: { width: { xs: '100%', sm: 140 } },
                }}
              />
              <SelectInput
                name="intervalPeriod"
                control={control}
                label="Period"
                formControlProps={{
                  size: 'small',
                  sx: { width: { xs: '100%', sm: 180 } },
                }}
              >
                <MenuItem value="days">days</MenuItem>
                <MenuItem value="hours">hours</MenuItem>
                <MenuItem value="minutes">minutes</MenuItem>
              </SelectInput>
            </Stack>

            <Box sx={{ maxWidth: 340 }}>
              <DateTimeInput
                name="startTime"
                control={control}
                label={`Start time (${INTERVAL_TIMEZONE})`}
              />
            </Box>
          </>
        ) : (
          <>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
              <Box sx={{ flex: 1 }}>
                <TextInput
                  name="cronExpression"
                  control={control}
                  label="Cron expression"
                  isRequired
                  controllerProps={{
                    rules: {
                      required: 'Enter a cron expression',
                      validate: (v) =>
                        humanize(String(v)).valid || 'Invalid cron expression',
                    },
                  }}
                  textFieldProps={{
                    size: 'small',
                    fullWidth: true,
                    placeholder: '*/5 * * * *',
                  }}
                />
              </Box>
              <Box sx={{ width: { xs: '100%', sm: 260 } }}>
                <AutoCompleteInput
                  name="cronTimezone"
                  control={control}
                  label="Timezone"
                  options={timezoneOptions}
                  autoCompleteProps={{
                    size: 'small',
                    disableClearable: true,
                  }}
                />
              </Box>
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
          </>
        )}

        <Box>
          <Typography
            variant="caption"
            color="text.secondary"
            data-testid="sched-form-timezone-notice"
          >
            {scheduleMode === 'cron'
              ? `Runs in ${zoneInForce}.`
              : `Runs in ${zoneInForce} — an interval schedule has no timezone of its own.`}
          </Typography>

          {previewSpec !== null && (
            <Typography
              variant="caption"
              color={previewFailed ? 'error' : 'text.secondary'}
              sx={{ display: 'block', mt: 0.5 }}
              data-testid="sched-form-next-runs"
            >
              {previewFailed
                ? 'Could not work out the next runs for this schedule.'
                : nextRuns.length > 0
                  ? `Next runs: ${nextRuns.map(formatNextRun).join(', ')}`
                  : preview
                    ? 'This schedule has no upcoming runs.'
                    : 'Working out the next runs…'}
            </Typography>
          )}
        </Box>

        <SwitchInput name="enabled" control={control} label="Enabled" />

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
      </Stack>
    </Box>
  );
}
