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
import { Controller, FormProvider, useForm, useWatch } from 'react-hook-form';
import {
  Alert,
  Box,
  CircularProgress,
  Dialog,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { useSnackbar } from 'notistack';
import {
  SchemaFormRenderer,
  ServiceSelector,
  TaskHistoryTable,
  TaskLogViewer,
  useStopTaskHistory,
  type ServiceOption,
  type TaskHistoryEntry,
} from '@sep/framework';
import {
  useDipperExecution,
  useDipperFormSchema,
  useDipperHistory,
  useDipperAppSchema,
} from './hooks';
import type { DipperCollectorType } from './types';

const RESERVED_EXECUTION_FIELDS = new Set(['executor_host', 'sudo', 'script_preview']);
const SERVICE_TYPES = ['mysql', 'mongodb', 'postgresql'] as const;

interface DipperSelectionValues {
  service_id: ServiceOption | null;
  collector_type: DipperCollectorType;
}

function selectedServiceId(service: ServiceOption | null | undefined): number | null {
  return typeof service?.id === 'number' ? service.id : null;
}

function buildArgs(values: Record<string, unknown>): Record<string, unknown> {
  const args: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(values)) {
    if (RESERVED_EXECUTION_FIELDS.has(key)) {
      continue;
    }
    if (value === '' || value === undefined || value === null) {
      continue;
    }
    args[key] = value;
  }
  return args;
}

export function DipperApp() {
  const { enqueueSnackbar } = useSnackbar();
  const selectionForm = useForm<DipperSelectionValues>({
    defaultValues: { service_id: null, collector_type: 'environment' },
  });
  const selectedService = useWatch({ control: selectionForm.control, name: 'service_id' });
  const collectorType = useWatch({ control: selectionForm.control, name: 'collector_type' });
  const serviceId = selectedServiceId(selectedService);

  const appSchema = useDipperAppSchema();
  const formSchema = useDipperFormSchema(serviceId, collectorType);
  const history = useDipperHistory();
  const execution = useDipperExecution();
  const stop = useStopTaskHistory();
  const [logsEntry, setLogsEntry] = useState<TaskHistoryEntry | null>(null);

  const title = appSchema.data?.display_name ?? 'Collect Diagnostic Data';
  const description =
    appSchema.data?.description ?? 'Run diagnostic data collection scripts on managed hosts.';

  const handleSubmit = (values: Record<string, unknown>) => {
    if (serviceId === null) {
      return;
    }
    execution.mutate(
      {
        service_id: serviceId,
        collector_type: collectorType,
        executor_host: String(values.executor_host ?? ''),
        sudo: Boolean(values.sudo ?? false),
        args: buildArgs(values),
      },
      {
        onSuccess: () => {
          enqueueSnackbar('Data collection started', { variant: 'success' });
        },
      },
    );
  };

  const submitError = useMemo(() => {
    if (!execution.isError) {
      return null;
    }
    return execution.error?.message ?? 'Execution failed';
  }, [execution.error, execution.isError]);

  const handleViewLogs = useCallback((entry: TaskHistoryEntry) => {
    setLogsEntry(entry);
  }, []);

  const handleCloseLogs = useCallback(() => {
    setLogsEntry(null);
  }, []);

  if (appSchema.isLoading && !appSchema.data) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (appSchema.error) {
    return <Alert severity="error">Failed to load Dipper schema: {appSchema.error.message}</Alert>;
  }

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 1 }}>
        {title}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        {description}
      </Typography>

      <FormProvider {...selectionForm}>
        <Box sx={{ maxWidth: 640, display: 'grid', gap: 2, mb: 3 }}>
          <ServiceSelector
            name="service_id"
            label="Database Service"
            required
            serviceTypes={SERVICE_TYPES}
          />
          <Controller
            name="collector_type"
            control={selectionForm.control}
            render={({ field }) => (
              <FormControl fullWidth required>
                <InputLabel id="dipper-collector-type-label">Collector Type</InputLabel>
                <Select {...field} labelId="dipper-collector-type-label" label="Collector Type">
                  <MenuItem value="environment">Environment</MenuItem>
                  <MenuItem value="pmm">PMM</MenuItem>
                </Select>
              </FormControl>
            )}
          />
        </Box>
      </FormProvider>

      {serviceId === null ? (
        <Alert severity="info" sx={{ maxWidth: 640 }}>
          Select a database service to load the execution form.
        </Alert>
      ) : formSchema.isLoading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', maxWidth: 640, py: 4 }}>
          <CircularProgress />
        </Box>
      ) : formSchema.error || !formSchema.data ? (
        <Alert severity="error">
          Failed to load Dipper form: {formSchema.error?.message ?? 'unknown error'}
        </Alert>
      ) : (
        <SchemaFormRenderer
          key={`${serviceId}-${collectorType}`}
          sections={formSchema.data.forms ?? []}
          onSubmit={handleSubmit}
          submitLabel="Execute"
          loading={execution.isPending}
          submitError={submitError}
        />
      )}

      <Divider sx={{ my: 4 }} />

      <Typography variant="h6" sx={{ mb: 1 }}>
        Execution history
      </Typography>
      {history.error ? (
        <Alert severity="error">Failed to load execution history: {history.error.message}</Alert>
      ) : (
        <TaskHistoryTable
          data={history.data?.items ?? []}
          isLoading={history.isLoading}
          onViewLogs={handleViewLogs}
          onStopTask={(entry) => {
            if (entry.id !== null && entry.id !== undefined) {
              // Dipper's history is keyed under ['dipper','history'] and does not
              // poll, so the stop hook's ['task-history'] invalidation never
              // reaches it; refetch this query directly once the stop succeeds.
              stop.mutate(entry.id, {
                onSuccess: () => {
                  history.refetch();
                },
              });
            }
          }}
          isStopping={stop.isPending}
        />
      )}

      <Dialog
        open={logsEntry !== null}
        onClose={handleCloseLogs}
        fullWidth
        maxWidth="lg"
        aria-labelledby="dipper-task-logs-title"
      >
        <DialogTitle
          id="dipper-task-logs-title"
          sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}
        >
          <span>
            Task logs
            {logsEntry?.task?.name ? ` - ${logsEntry.task.name}` : ''}
            {logsEntry?.id !== null && logsEntry?.id !== undefined ? ` #${logsEntry.id}` : ''}
          </span>
          <IconButton aria-label="Close logs dialog" onClick={handleCloseLogs} size="small">
            <CloseIcon fontSize="small" />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers sx={{ p: 0 }}>
          {logsEntry?.id !== null && logsEntry?.id !== undefined ? (
            <TaskLogViewer
              taskHistoryId={logsEntry.id}
              taskStatus={logsEntry.status}
              height={520}
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </Box>
  );
}
