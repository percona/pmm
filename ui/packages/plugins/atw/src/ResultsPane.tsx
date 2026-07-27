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
  Button,
  Chip,
  CircularProgress,
  Divider,
  Stack,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import FolderOpenIcon from '@mui/icons-material/FolderOpen';
import {
  TaskFilesDialog,
  TaskHistoryStatusBadge,
  TaskLogViewer,
  isTaskHistoryStatus,
} from '@sep/framework';
import { useAtwIncidentExecutions } from './hooks';
import type { AtwIncidentExecution } from './types';

export interface ResultsPaneProps {
  incidentId: string;
}

/**
 * The Results pane: lists the incident's executions, each with its status, logs,
 * and downloadable file listing.
 *
 * This is the pane shell only. A future "send to ServiceNow" action
 * (execution selection, case-ref prefill, job progress) will add a cross-row
 * selection model and an action bar here — row state currently lives in
 * {@link ExecutionRow} and will need lifting when that lands.
 */
export function ResultsPane({ incidentId }: ResultsPaneProps) {
  const { data, isLoading, error } = useAtwIncidentExecutions(incidentId);
  const [filesForTask, setFilesForTask] = useState<number | null>(null);

  return (
    <Box>
      <Typography variant="h6" sx={{ mb: 2 }}>
        Results
      </Typography>

      {isLoading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      )}

      {error && (
        <Alert severity="error">
          Failed to load executions: {error.message}
        </Alert>
      )}

      {!isLoading && !error && (!data || data.length === 0) && (
        <Alert severity="info">
          No executions yet. Run snippets from the Collect pane to see results
          here.
        </Alert>
      )}

      {data?.map((execution) => (
        <ExecutionRow
          key={execution.id}
          execution={execution}
          onOpenFiles={() => setFilesForTask(execution.task_history_id)}
        />
      ))}

      <TaskFilesDialog
        open={filesForTask !== null}
        taskHistoryId={filesForTask}
        onClose={() => setFilesForTask(null)}
      />
    </Box>
  );
}

function ExecutionRow({
  execution,
  onOpenFiles,
}: {
  execution: AtwIncidentExecution;
  onOpenFiles: () => void;
}) {
  const { snippet_filename, task_status, task_history_id, has_logs } =
    execution;

  return (
    <Accordion
      disableGutters
      sx={{ mb: 1 }}
      slotProps={{ transition: { unmountOnExit: true } }}
    >
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Stack
          direction="row"
          spacing={1}
          alignItems="center"
          sx={{ width: '100%', pr: 1, flexWrap: 'wrap' }}
        >
          <Typography
            variant="subtitle2"
            sx={{ flexGrow: 1, wordBreak: 'break-all' }}
          >
            {snippet_filename}
          </Typography>
          {isTaskHistoryStatus(task_status) ? (
            <TaskHistoryStatusBadge status={task_status} />
          ) : (
            <Chip size="small" label="Unknown" />
          )}
        </Stack>
      </AccordionSummary>
      <AccordionDetails
        sx={(theme) => ({ paddingRight: theme.spacing(2) + ' !important' })}
      >
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <Button
            size="small"
            variant="outlined"
            startIcon={<FolderOpenIcon />}
            onClick={onOpenFiles}
          >
            Files
          </Button>
        </Stack>

        <Divider sx={{ mb: 2 }} />

        {has_logs === false ? (
          <Typography variant="body2" color="text.secondary">
            No logs available for this execution.
          </Typography>
        ) : (
          <TaskLogViewer
            taskHistoryId={task_history_id}
            taskStatus={task_status ?? undefined}
            height={360}
          />
        )}
      </AccordionDetails>
    </Accordion>
  );
}
