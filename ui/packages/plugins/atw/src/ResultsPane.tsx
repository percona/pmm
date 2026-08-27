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
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  Button,
  Checkbox,
  Chip,
  CircularProgress,
  Divider,
  Paper,
  Stack,
  TablePagination,
  Tooltip,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import FolderOpenIcon from '@mui/icons-material/FolderOpen';
import SendIcon from '@mui/icons-material/Send';
import { useAuth } from '@sep/api';
import {
  TaskFilesDialog,
  TaskHistoryStatusBadge,
  TaskLogViewer,
  isTaskHistoryStatus,
} from '@sep/framework';
import {
  ATW_PAGE_SIZE,
  sendJobDetail,
  useAtwConfig,
  useAtwIncident,
  useAtwIncidentExecutions,
  useAtwSendJobs,
} from './hooks';
import { SendDialog } from './SendDialog';
import type {
  AtwIncidentExecution,
  AtwSendLog,
  AtwSendLogExecution,
} from './types';

export interface ResultsPaneProps {
  incidentId: string;
}

/** What a Re-send replays: the attempt's own executions and case reference. */
interface ResendContext {
  executions: AtwSendLogExecution[];
  caseRef: string;
}

/**
 * Task statuses whose execution is finished and therefore sendable.
 *
 * Mirrors the backend's own `TaskHistoryStatusEnum.is_finished()`; typing it
 * against the generated status union keeps a renamed or added status a compile
 * error rather than a silently unselectable row.
 */
const FINISHED_TASK_STATUSES: ReadonlySet<
  NonNullable<AtwIncidentExecution['task_status']>
> = new Set(['success', 'failed', 'stopped', 'stale']);

const SEND_STATUS_COLORS = {
  success: 'success',
  failed: 'error',
  running: 'info',
  pending: 'default',
} as const;

function isSelectable(execution: AtwIncidentExecution): boolean {
  const status = execution.task_status;
  return (
    status !== null &&
    status !== undefined &&
    FINISHED_TASK_STATUSES.has(status)
  );
}

/**
 * The Results pane: lists the incident's executions with their status, logs and
 * files, and lets a support engineer send a selection to the support case.
 *
 * Selection is keyed by execution id and held above the row list, so it survives
 * a page flip — deriving it from the rendered rows would silently drop anything
 * chosen on an earlier page.
 */
export function ResultsPane({ incidentId }: ResultsPaneProps) {
  const { canMutate } = useAuth();
  const [page, setPage] = useState({ offset: 0, limit: ATW_PAGE_SIZE });
  const { data, isLoading, error } = useAtwIncidentExecutions(incidentId, page);
  const { data: incident } = useAtwIncident(incidentId);
  const { data: config } = useAtwConfig();
  const { data: sendJobs, error: sendJobsError } = useAtwSendJobs(incidentId);

  const [filesForTask, setFilesForTask] = useState<number | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [sendOpen, setSendOpen] = useState(false);
  const [sendSessionKey, setSendSessionKey] = useState(0);
  const [resend, setResend] = useState<ResendContext | null>(null);

  useEffect(() => {
    if (data && data.total > 0 && data.offset >= data.total) {
      setPage((previous) => ({
        offset: Math.max(
          0,
          (Math.ceil(data.total / previous.limit) - 1) * previous.limit
        ),
        limit: previous.limit,
      }));
    }
  }, [data]);

  const rows = data?.items;
  const knownExecutions = useMemo(() => {
    const map = new Map<string, AtwSendLogExecution>();
    for (const execution of rows ?? []) {
      map.set(execution.id, {
        id: execution.id,
        task_history_id: execution.task_history_id,
        snippet_filename: execution.snippet_filename,
      });
    }
    return map;
  }, [rows]);

  const [selectionLabels, setSelectionLabels] = useState<
    Map<string, AtwSendLogExecution>
  >(new Map());

  const toggleSelected = (execution: AtwIncidentExecution) => {
    setSelectedIds((previous) => {
      const next = new Set(previous);
      if (next.has(execution.id)) {
        next.delete(execution.id);
      } else {
        next.add(execution.id);
      }
      return next;
    });
    setSelectionLabels((previous) => {
      const next = new Map(previous);
      next.set(execution.id, {
        id: execution.id,
        task_history_id: execution.task_history_id,
        snippet_filename: execution.snippet_filename,
      });
      return next;
    });
  };

  /**
   * The finished executions on the page currently rendered.
   *
   * Reusing `isSelectable` is what keeps the header toggle and the row
   * checkboxes from ever disagreeing about which rows are eligible.
   */
  const selectablePageIds = useMemo(
    () => (rows ?? []).filter(isSelectable).map((execution) => execution.id),
    [rows]
  );

  const selectedPageCount = selectablePageIds.filter((id) =>
    selectedIds.has(id)
  ).length;
  const allPageSelected =
    selectablePageIds.length > 0 &&
    selectedPageCount === selectablePageIds.length;
  const somePageSelected = selectedPageCount > 0 && !allPageSelected;

  /**
   * Select or deselect the page's finished executions in one action.
   *
   * Deselection removes only this page's ids: the selection deliberately
   * outlives a page flip, so clearing the whole set would drop rows chosen on
   * another page.
   */
  const toggleSelectAllOnPage = () => {
    const deselecting = allPageSelected;
    setSelectedIds((previous) => {
      const next = new Set(previous);
      for (const id of selectablePageIds) {
        if (deselecting) {
          next.delete(id);
        } else {
          next.add(id);
        }
      }
      return next;
    });
    if (deselecting) {
      return;
    }
    // Mirror the labels the send dialog needs, so a row the user later pages
    // away from still names itself. `knownExecutions` already holds them in the
    // shape `toggleSelected` writes.
    setSelectionLabels((previous) => {
      const next = new Map(previous);
      for (const id of selectablePageIds) {
        const execution = knownExecutions.get(id);
        if (execution) {
          next.set(id, execution);
        }
      }
      return next;
    });
  };

  const selectedExecutions = [...selectedIds]
    .map((id) => knownExecutions.get(id) ?? selectionLabels.get(id))
    .filter(
      (execution): execution is AtwSendLogExecution => execution !== undefined
    );

  const disabledReasons = config?.send_disabled_reasons ?? [];
  const sendDisabled =
    selectedExecutions.length === 0 || disabledReasons.length > 0;
  const sendTooltip = disabledReasons.length
    ? disabledReasons.join('; ')
    : selectedExecutions.length === 0
      ? 'Select one or more finished executions to send.'
      : '';

  const openSend = (context: ResendContext | null) => {
    setResend(context);
    setSendSessionKey((previous) => previous + 1);
    setSendOpen(true);
  };

  const closeSend = (started: boolean) => {
    setSendOpen(false);
    if (started && resend === null) {
      setSelectedIds(new Set());
      setSelectionLabels(new Map());
    }
  };

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

      {!isLoading && !error && (!rows || rows.length === 0) && (
        <Alert severity="info">
          {canMutate
            ? 'No executions yet. Run snippets from the Collect pane to see results here.'
            : 'No executions yet.'}
        </Alert>
      )}

      {/* Selection exists only to feed the send action, so both go together. */}
      {rows && rows.length > 0 && canMutate && (
        <Stack
          direction="row"
          spacing={2}
          alignItems="center"
          sx={{ mb: 2, flexWrap: 'wrap', rowGap: 1 }}
        >
          <Tooltip
            title={
              selectablePageIds.length === 0
                ? 'This page has no finished executions to select.'
                : ''
            }
          >
            <span>
              <Checkbox
                size="small"
                checked={allPageSelected}
                indeterminate={somePageSelected}
                disabled={selectablePageIds.length === 0}
                onChange={toggleSelectAllOnPage}
                inputProps={{
                  'aria-label': 'Select all finished executions on this page',
                }}
              />
            </span>
          </Tooltip>
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ flexGrow: 1 }}
          >
            {selectedExecutions.length} selected
          </Typography>
          <Tooltip title={sendTooltip}>
            <span>
              <Button
                variant="contained"
                startIcon={<SendIcon />}
                disabled={sendDisabled}
                onClick={() => openSend(null)}
              >
                Send to support case
              </Button>
            </span>
          </Tooltip>
        </Stack>
      )}

      {rows?.map((execution) => (
        <ExecutionRow
          key={execution.id}
          execution={execution}
          selected={selectedIds.has(execution.id)}
          onToggleSelected={() => toggleSelected(execution)}
          onOpenFiles={() => setFilesForTask(execution.task_history_id)}
        />
      ))}

      {data && data.total > data.limit && (
        <TablePagination
          component="div"
          count={data.total}
          page={Math.floor(data.offset / Math.max(data.limit, 1))}
          rowsPerPage={data.limit}
          onPageChange={(_event, newPage) =>
            setPage((previous) => ({
              offset: newPage * previous.limit,
              limit: previous.limit,
            }))
          }
          rowsPerPageOptions={[ATW_PAGE_SIZE]}
        />
      )}

      <SendHistory
        jobs={sendJobs?.items}
        total={sendJobs?.total}
        error={sendJobsError}
        onResend={openSend}
        disabledReasons={disabledReasons}
      />

      <TaskFilesDialog
        open={filesForTask !== null}
        taskHistoryId={filesForTask}
        onClose={() => setFilesForTask(null)}
      />

      <SendDialog
        key={sendSessionKey}
        open={sendOpen}
        incidentId={incidentId}
        executions={resend?.executions ?? selectedExecutions}
        defaultCaseRef={resend?.caseRef ?? incident?.case_ref}
        onClose={closeSend}
      />
    </Box>
  );
}

/**
 * Past send attempts for this incident, with Re-send on the failed ones.
 *
 * A re-send replays the attempt's own recorded executions and case reference.
 * Executions deleted since the attempt are rejected by the POST, which names
 * them — the pane cannot filter them itself, because it only ever holds one
 * page of executions and would drop every id that happens to sit on another.
 */
function SendHistory({
  jobs,
  total,
  error,
  onResend,
  disabledReasons,
}: {
  jobs: AtwSendLog[] | undefined;
  total: number | undefined;
  error: Error | null;
  onResend: (context: ResendContext) => void;
  disabledReasons: string[];
}) {
  const { canMutate } = useAuth();
  const resendDisabled = disabledReasons.length > 0;
  const resendTooltip = disabledReasons.join('; ');

  if (error) {
    return (
      <Alert severity="error" sx={{ mt: 3 }}>
        Could not load the send history: {error.message}
      </Alert>
    );
  }

  if (!jobs || jobs.length === 0) {
    return null;
  }

  return (
    <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
      <Typography variant="subtitle2" sx={{ mb: 1 }}>
        Send history
      </Typography>
      <Stack divider={<Divider flexItem />} spacing={1}>
        {jobs.map((job) => {
          const detail = sendJobDetail(job);
          return (
            <Stack
              key={job.id}
              direction="row"
              spacing={1}
              alignItems="center"
              sx={{ flexWrap: 'wrap', rowGap: 1 }}
            >
              <Chip
                size="small"
                label={job.status}
                color={SEND_STATUS_COLORS[job.status]}
              />
              <Typography variant="body2" sx={{ flexGrow: 1 }}>
                {job.case_ref} · {job.requested_by}
                {job.finished_at
                  ? ` · ${new Date(job.finished_at).toLocaleString()}`
                  : ''}
              </Typography>
              {job.status === 'failed' && canMutate && (
                <Tooltip title={resendTooltip}>
                  <span>
                    <Button
                      size="small"
                      disabled={resendDisabled}
                      onClick={() =>
                        onResend({
                          executions: detail.executions ?? [],
                          caseRef: job.case_ref,
                        })
                      }
                    >
                      Re-send
                    </Button>
                  </span>
                </Tooltip>
              )}
              {job.status === 'failed' && detail.error && (
                <Typography
                  variant="body2"
                  color="error"
                  sx={{ width: '100%' }}
                >
                  {detail.error}
                </Typography>
              )}
            </Stack>
          );
        })}
      </Stack>
      {total !== undefined && total > jobs.length && (
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ display: 'block', mt: 1 }}
        >
          Showing the {jobs.length} most recent of {total} attempts.
        </Typography>
      )}
    </Paper>
  );
}

function ExecutionRow({
  execution,
  selected,
  onToggleSelected,
  onOpenFiles,
}: {
  execution: AtwIncidentExecution;
  selected: boolean;
  onToggleSelected: () => void;
  onOpenFiles: () => void;
}) {
  const { canMutate } = useAuth();
  const {
    snippet_filename,
    task_status,
    task_history_id,
    has_logs,
    masked_args,
    args_withheld,
  } = execution;
  const selectable = isSelectable(execution);

  return (
    <Accordion
      disableGutters
      sx={{ mb: 1 }}
      slotProps={{ transition: { unmountOnExit: true } }}
    >
      <AccordionSummary
        expandIcon={<ExpandMoreIcon />}
        sx={{ '& .MuiAccordionSummary-content': { minWidth: 0 } }}
      >
        <Stack
          direction="row"
          spacing={1}
          alignItems="center"
          sx={{ width: '100%', pr: 1, flexWrap: 'wrap' }}
        >
          {/* Selection exists only to feed the send action, so both go together. */}
          {canMutate && (
            <Tooltip
              title={selectable ? '' : 'Only finished executions can be sent.'}
            >
              <span>
                <Checkbox
                  size="small"
                  checked={selected}
                  disabled={!selectable}
                  onChange={onToggleSelected}
                  onClick={(event) => event.stopPropagation()}
                  inputProps={{ 'aria-label': `Select ${snippet_filename}` }}
                />
              </span>
            </Tooltip>
          )}
          <Box sx={{ flexGrow: 1, minWidth: 0 }}>
            <Typography variant="subtitle2" sx={{ wordBreak: 'break-all' }}>
              {snippet_filename}
            </Typography>
            {masked_args && (
              <Typography
                variant="body2"
                color="text.secondary"
                sx={{
                  fontFamily: 'monospace',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {masked_args}
              </Typography>
            )}
          </Box>
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

        <Box sx={{ mb: 2 }}>
          <Typography
            variant="caption"
            color="text.secondary"
            sx={{ display: 'block' }}
          >
            Arguments
          </Typography>
          {masked_args ? (
            <Typography
              variant="body2"
              color="text.secondary"
              sx={{
                fontFamily: 'monospace',
                wordBreak: 'break-all',
                whiteSpace: 'pre-wrap',
              }}
            >
              {masked_args}
            </Typography>
          ) : (
            <Typography variant="body2" color="text.secondary">
              {args_withheld ? 'Arguments unavailable' : 'No arguments'}
            </Typography>
          )}
        </Box>

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
