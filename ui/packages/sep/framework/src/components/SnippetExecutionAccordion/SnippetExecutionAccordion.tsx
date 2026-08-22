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
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  CircularProgress,
  Dialog,
  DialogContent,
  DialogTitle,
  Divider,
  IconButton,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient, useAuth, type PluginSchema } from '@sep/api';
import {
  snippetPluginExecutePath,
  snippetPluginHistoryPath,
  snippetPluginSchemaPath,
} from '../../snippetPluginPaths';
import { ReadOnlyNotice } from '../ReadOnlyNotice';
import { SchemaFormRenderer } from '../SchemaFormRenderer';
import {
  TaskHistoryTable,
  type PaginatedTaskHistory,
  type TaskHistoryEntry,
} from '../TaskHistoryTable';
import { TaskLogViewer } from '../TaskLogViewer';
import { useStopTaskHistory } from '../../hooks';

/** Fields excluded from the snippet `args` payload (handled at top level instead). */
const ARGS_EXCLUDED_FIELDS = new Set([
  'executor_host',
  'sudo',
  'script_preview',
]);

export interface SnippetExecutionAccordionProps {
  snippetFilename: string;
  /** When non-empty, strips ``executor_host`` from the rendered form and injects this value at submit. */
  executorHost?: string;
  title?: string;
  description?: string;
  defaultExpanded?: boolean;
  showHistory?: boolean;
}

interface ExecuteResponse {
  task_id: number | null;
}

interface SnippetExecutionRequest {
  executor_host: string;
  sudo: boolean;
  args: Record<string, unknown>;
}

function useSnippetAccordionSchema(filename: string, enabled: boolean) {
  return useQuery<PluginSchema>({
    queryKey: ['snippets', filename, 'schema', { execution_only: true }],
    queryFn: async () => {
      const { data } = await apiClient.get<PluginSchema>(
        snippetPluginSchemaPath(filename),
        {
          params: { execution_only: true },
        }
      );
      return data;
    },
    enabled,
    staleTime: Infinity,
  });
}

function useSnippetAccordionExecution(filename: string) {
  const queryClient = useQueryClient();
  return useMutation<ExecuteResponse, Error, SnippetExecutionRequest>({
    mutationFn: async (body) => {
      const { data } = await apiClient.post<ExecuteResponse>(
        snippetPluginExecutePath(filename),
        body
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['snippets', filename, 'history'],
      });
    },
  });
}

function useSnippetAccordionHistory(filename: string, enabled: boolean) {
  return useQuery<PaginatedTaskHistory>({
    queryKey: ['snippets', filename, 'history'],
    queryFn: async () => {
      const { data } = await apiClient.get<PaginatedTaskHistory>(
        snippetPluginHistoryPath(filename)
      );
      return data;
    },
    enabled,
  });
}

/**
 * Self-contained accordion card that renders a snippet form, executes it,
 * and shows the live log output. Intended for multi-snippet pages (alert
 * troubleshooting) where one card per snippet is mounted on the same page.
 *
 * Schema fetch is deferred until the accordion is expanded. When a non-empty
 * ``executorHost`` is provided, ``executor_host`` is stripped from the rendered
 * form and injected at submit time — a single shared host selector drives all
 * cards on the page. When omitted or empty, the field renders normally from
 * the schema so the user selects a host per-card.
 *
 * When ``showHistory`` is ``true`` (used by the snippets detail page), a
 * per-task execution history table is rendered below the form.
 */
export function SnippetExecutionAccordion({
  snippetFilename,
  executorHost,
  title,
  description,
  defaultExpanded = false,
  showHistory = false,
}: SnippetExecutionAccordionProps) {
  const { canMutate } = useAuth();
  const [expanded, setExpanded] = useState(defaultExpanded);
  const [currentTaskId, setCurrentTaskId] = useState<number | null>(null);
  const [logsEntry, setLogsEntry] = useState<TaskHistoryEntry | null>(null);

  // The form is the execute control, so a read-only session never renders it —
  // and never needs its schema. Disabling the query is only the request
  // optimization: react-query still serves a cached entry, and this schema is
  // held with `staleTime: Infinity` under a key that carries no identity, so an
  // admin's fetch would otherwise render the form for a non-admin reaching the
  // same snippet later in the same tab. The render gates on `canMutate` too.
  // History and logs stay readable.
  const schemaQuery = useSnippetAccordionSchema(
    snippetFilename,
    expanded && canMutate
  );
  const executionMutation = useSnippetAccordionExecution(snippetFilename);
  const historyQuery = useSnippetAccordionHistory(snippetFilename, showHistory);
  const stop = useStopTaskHistory();

  const displayTitle = title ?? snippetFilename;

  const hoistingHost = Boolean(executorHost);
  const filteredSections = (schemaQuery.data?.forms ?? []).map((section) => ({
    ...section,
    fields: section.fields.filter(
      (field) => field.name !== 'executor_host' || !hoistingHost
    ),
  }));

  const handleSubmit = (values: Record<string, unknown>) => {
    const args: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(values)) {
      if (ARGS_EXCLUDED_FIELDS.has(key)) {
        continue;
      }
      if (value === '' || value === undefined) {
        continue;
      }
      args[key] = value;
    }
    executionMutation.mutate(
      {
        executor_host: String(
          hoistingHost ? executorHost : (values.executor_host ?? '')
        ),
        sudo: Boolean(values.sudo ?? false),
        args,
      },
      {
        onSuccess: (data) => {
          if (data.task_id !== null && data.task_id !== undefined) {
            setCurrentTaskId(data.task_id);
          }
        },
      }
    );
  };

  const submitError = executionMutation.isError
    ? (executionMutation.error?.message ?? 'Execution failed')
    : null;

  return (
    <Accordion
      expanded={expanded}
      onChange={(_, isExpanded) => setExpanded(isExpanded)}
      disableGutters
    >
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Box>
          <Typography variant="subtitle1" fontWeight={500}>
            {displayTitle}
          </Typography>
          {description && (
            <Typography variant="body2" color="text.secondary">
              {description}
            </Typography>
          )}
        </Box>
      </AccordionSummary>

      <AccordionDetails>
        {!canMutate && (
          <ReadOnlyNotice
            variant="inline"
            action="execute this snippet"
            testId="snippet-execute-read-only"
          />
        )}

        {schemaQuery.isLoading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
            <CircularProgress size={24} />
          </Box>
        )}

        {schemaQuery.isError && (
          <Alert severity="error">
            Failed to load form: {schemaQuery.error?.message ?? 'unknown error'}
          </Alert>
        )}

        {canMutate && schemaQuery.data && (
          <SchemaFormRenderer
            sections={filteredSections}
            onSubmit={handleSubmit}
            submitLabel="Execute"
            loading={executionMutation.isPending}
            submitError={submitError}
          />
        )}

        {currentTaskId !== null && (
          <Box sx={{ mt: 2 }}>
            <Divider sx={{ mb: 2 }} />
            <TaskLogViewer taskHistoryId={currentTaskId} />
          </Box>
        )}

        {showHistory && (
          <>
            <Divider sx={{ my: 3 }} />
            <Typography variant="h6" sx={{ mb: 1 }}>
              Execution history
            </Typography>
            {historyQuery.error ? (
              <Alert severity="error">
                Failed to load execution history: {historyQuery.error.message}
              </Alert>
            ) : (
              <TaskHistoryTable
                data={historyQuery.data?.items ?? []}
                isLoading={historyQuery.isLoading}
                hideTaskNameColumn
                onViewLogs={setLogsEntry}
                onStopTask={(entry) => {
                  if (entry.id !== null && entry.id !== undefined) {
                    // This accordion's history is keyed under
                    // ['snippets', filename, 'history'] and does not poll, so the
                    // stop hook's ['task-history'] invalidation never reaches it;
                    // refetch this query directly once the stop succeeds.
                    stop.mutate(entry.id, {
                      onSuccess: () => {
                        historyQuery.refetch();
                      },
                    });
                  }
                }}
                isStopping={stop.isPending}
              />
            )}
          </>
        )}
      </AccordionDetails>

      <Dialog
        open={logsEntry !== null}
        onClose={() => setLogsEntry(null)}
        fullWidth
        maxWidth="lg"
        aria-labelledby="snippet-task-logs-title"
      >
        <DialogTitle
          id="snippet-task-logs-title"
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <span>
            Task logs
            {logsEntry?.task?.name ? ` - ${logsEntry.task.name}` : ''}
            {logsEntry?.id !== null && logsEntry?.id !== undefined
              ? ` #${logsEntry.id}`
              : ''}
          </span>
          <IconButton
            aria-label="Close logs dialog"
            onClick={() => setLogsEntry(null)}
            size="small"
          >
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
    </Accordion>
  );
}
