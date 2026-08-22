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
import DownloadIcon from '@mui/icons-material/Download';
import StopCircleIcon from '@mui/icons-material/StopCircle';
import VisibilityIcon from '@mui/icons-material/Visibility';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { MaterialReactTable, type MRT_ColumnDef } from 'material-react-table';
import { useAuth } from '@sep/api';
import {
  isRunningStatus,
  useStopTaskHistory,
  useTaskHistory,
  useTaskHistoryByName,
} from '../../hooks/useTaskHistory';
import { useTaskHistoryFiles } from '../../hooks/useTaskHistoryFiles';
import { SEP_TABLE_CLASS } from '../../constants';
import { ChainDisplay } from './ChainDisplay';
import { StatusBadge } from './StatusBadge';
import { TaskFilesDialog } from './TaskFilesDialog';
import type {
  TaskHistoryEntry,
  TaskHistoryTableProps,
} from './TaskHistoryTable.types';

/** Cache file-list probes across history-table poll ticks. */
const DOWNLOADABLE_FILES_STALE_TIME_MS = 30_000;

function formatDateTime(value?: string | null): string {
  if (!value) {
    return '—';
  }
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString();
}

function formatDuration(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined) {
    return '—';
  }
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`;
  }
  const total = Math.round(seconds);
  const m = Math.floor(total / 60);
  const s = total % 60;
  return `${m}m ${s}s`;
}

interface MetaShape {
  _chain_task_names?: string[];
  _chain_depth?: number;
}

function readMeta(entry: TaskHistoryEntry): MetaShape {
  const meta = entry.execution_request?.meta as MetaShape | null | undefined;
  return meta ?? {};
}

function canProbeDownloadableFiles(entry: TaskHistoryEntry): boolean {
  return (
    !isRunningStatus(entry.status) &&
    entry.id !== null &&
    entry.id !== undefined &&
    Boolean(entry.task?.output_files_path)
  );
}

function stopConfirmLabel(entry: TaskHistoryEntry): string {
  return entry.task?.name ?? `#${entry.id ?? ''}`;
}

interface DownloadFilesButtonProps {
  entry: TaskHistoryEntry;
  onDownloadFiles: TaskHistoryTableProps['onDownloadFiles'];
  onOpenBuiltIn: (entry: TaskHistoryEntry) => void;
}

/**
 * Render the Download files action only when the run has user-visible files.
 *
 * ``has_logs`` is the wrong signal: logs can exist when the output directory
 * is empty (e.g. only the hidden ``.sep-run-result.json`` marker). Probe the
 * files API and hide the button until a non-empty listing is confirmed.
 */
function DownloadFilesButton({
  entry,
  onDownloadFiles,
  onOpenBuiltIn,
}: DownloadFilesButtonProps) {
  const shouldProbe = canProbeDownloadableFiles(entry);
  const { data, isLoading, isError } = useTaskHistoryFiles(
    shouldProbe ? entry.id : null,
    {
      staleTime: DOWNLOADABLE_FILES_STALE_TIME_MS,
    }
  );

  if (
    !shouldProbe ||
    isLoading ||
    isError ||
    !data ||
    Object.keys(data).length === 0
  ) {
    return null;
  }

  return (
    <Tooltip title="Download files">
      <span>
        <IconButton
          size="small"
          aria-label="Download files"
          onClick={() => {
            if (onDownloadFiles) {
              onDownloadFiles(entry);
            } else {
              onOpenBuiltIn(entry);
            }
          }}
        >
          <DownloadIcon fontSize="small" />
        </IconButton>
      </span>
    </Tooltip>
  );
}

interface ViewProps {
  rows: TaskHistoryEntry[];
  isLoading: boolean;
  resolveUserName: TaskHistoryTableProps['resolveUserName'];
  onViewLogs: TaskHistoryTableProps['onViewLogs'];
  onDownloadFiles: TaskHistoryTableProps['onDownloadFiles'];
  onChainItemClick: TaskHistoryTableProps['onChainItemClick'];
  hideTaskNameColumn?: boolean;
  /** Called once the user confirms the stop dialog. */
  onConfirmStop: (entry: TaskHistoryEntry) => void;
  isStopping: boolean;
  /** True when the row's stop action will resolve to a real handler (callback or internal mutation). */
  canStop: (entry: TaskHistoryEntry) => boolean;
}

function TaskHistoryTableView({
  rows,
  isLoading,
  resolveUserName,
  onViewLogs,
  onDownloadFiles,
  onChainItemClick,
  hideTaskNameColumn,
  onConfirmStop,
  isStopping,
  canStop,
}: ViewProps) {
  const { canMutate } = useAuth();
  const [pendingStopEntry, setPendingStopEntry] =
    useState<TaskHistoryEntry | null>(null);
  const [pendingFilesEntry, setPendingFilesEntry] =
    useState<TaskHistoryEntry | null>(null);

  const requestStop = useCallback(
    (entry: TaskHistoryEntry) => setPendingStopEntry(entry),
    []
  );
  const cancelStop = useCallback(() => setPendingStopEntry(null), []);
  const confirmStop = useCallback(() => {
    if (pendingStopEntry) {
      onConfirmStop(pendingStopEntry);
    }
    setPendingStopEntry(null);
  }, [onConfirmStop, pendingStopEntry]);
  const columns = useMemo<MRT_ColumnDef<TaskHistoryEntry>[]>(() => {
    const cols: MRT_ColumnDef<TaskHistoryEntry>[] = [
      {
        id: 'status',
        header: 'Status',
        size: 100,
        accessorFn: (row) => row.status,
        Cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      ...(hideTaskNameColumn
        ? []
        : [
            {
              id: 'task',
              header: 'Task',
              accessorFn: (row: TaskHistoryEntry) => row.task?.name ?? '',
              size: 140,
            } satisfies MRT_ColumnDef<TaskHistoryEntry>,
          ]),
      {
        id: 'host',
        header: 'Host',
        accessorFn: (row) => row.execution_request?.target ?? '',
        size: 140,
      },
      {
        id: 'chain',
        header: 'Chain',
        enableSorting: false,
        size: 150,
        Cell: ({ row }) => {
          const meta = readMeta(row.original);
          return (
            <ChainDisplay
              chainNames={meta._chain_task_names}
              chainDepth={meta._chain_depth}
              onChainItemClick={
                onChainItemClick
                  ? (name, index) => onChainItemClick(name, index, row.original)
                  : undefined
              }
            />
          );
        },
      },
      {
        id: 'started_at',
        header: 'Started',
        size: 160,
        accessorFn: (row) => row.started_at ?? '',
        sortingFn: 'datetime',
        Cell: ({ row }) => (
          <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
            {formatDateTime(row.original.started_at)}
          </Typography>
        ),
      },
      {
        id: 'duration',
        header: 'Duration',
        size: 90,
        accessorFn: (row) => row.duration ?? -1,
        Cell: ({ row }) => (
          <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
            {formatDuration(row.original.duration)}
          </Typography>
        ),
      },
      {
        id: 'executed_by',
        header: 'Executed By',
        size: 120,
        accessorFn: (row) =>
          (resolveUserName
            ? resolveUserName(row.executed_by)
            : (row.executed_by ?? '')) || '',
      },
      {
        id: 'actions',
        header: 'Actions',
        enableSorting: false,
        enableColumnFilter: false,
        size: 120,
        Cell: ({ row }) => {
          const entry = row.original;
          const running = isRunningStatus(entry.status);
          return (
            <Stack direction="row" spacing={0.5}>
              <Tooltip title="View logs">
                <span>
                  <IconButton
                    size="small"
                    aria-label="View logs"
                    disabled={!onViewLogs || (!entry.has_logs && !running)}
                    onClick={() => onViewLogs?.(entry)}
                  >
                    <VisibilityIcon fontSize="small" />
                  </IconButton>
                </span>
              </Tooltip>
              {running && canMutate && (
                <Tooltip title="Stop task">
                  <span>
                    <IconButton
                      size="small"
                      color="warning"
                      aria-label="Stop task"
                      onClick={() => requestStop(entry)}
                      disabled={isStopping || !canStop(entry)}
                    >
                      <StopCircleIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
              )}
              <DownloadFilesButton
                entry={entry}
                onDownloadFiles={onDownloadFiles}
                onOpenBuiltIn={setPendingFilesEntry}
              />
            </Stack>
          );
        },
      },
    ];
    return cols;
  }, [
    canMutate,
    hideTaskNameColumn,
    onChainItemClick,
    onDownloadFiles,
    onViewLogs,
    requestStop,
    isStopping,
    canStop,
    resolveUserName,
  ]);

  return (
    <>
      <MaterialReactTable
        columns={columns}
        data={rows}
        state={{ isLoading }}
        enableColumnActions={false}
        enableDensityToggle={false}
        enableFullScreenToggle={false}
        enableHiding={false}
        enablePagination
        enableSorting
        muiTablePaperProps={{
          className: SEP_TABLE_CLASS,
        }}
        initialState={{
          density: 'compact',
          sorting: [{ id: 'started_at', desc: true }],
          pagination: { pageIndex: 0, pageSize: 10 },
        }}
        getRowId={(row, index) =>
          String(
            row.id ?? `${row.task?.name ?? 'row'}-${row.started_at ?? index}`
          )
        }
        muiTableBodyRowProps={({ row }) =>
          isRunningStatus(row.original.status)
            ? {
                'data-running': 'true',
                sx: { backgroundColor: 'action.hover' },
              }
            : {}
        }
        renderEmptyRowsFallback={() => (
          <Typography
            variant="body2"
            sx={{ p: 2, textAlign: 'center' }}
            color="text.secondary"
          >
            No task history
          </Typography>
        )}
      />
      <Dialog
        open={pendingStopEntry !== null}
        onClose={cancelStop}
        aria-labelledby="task-history-stop-title"
      >
        <DialogTitle id="task-history-stop-title">Stop task</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {pendingStopEntry
              ? `Are you sure you want to stop the task ${stopConfirmLabel(pendingStopEntry)}?`
              : ''}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={cancelStop}>Cancel</Button>
          <Button onClick={confirmStop} variant="contained" autoFocus>
            Stop
          </Button>
        </DialogActions>
      </Dialog>
      <TaskFilesDialog
        open={pendingFilesEntry !== null}
        taskHistoryId={pendingFilesEntry?.id ?? null}
        onClose={() => setPendingFilesEntry(null)}
      />
    </>
  );
}

type ConnectedProps = Omit<TaskHistoryTableProps, 'data' | 'isLoading'>;

function ConnectedTaskHistoryTable({
  taskName,
  statusFilter,
  pollingIntervalMs,
  disablePolling,
  resolveUserName,
  onViewLogs,
  onStopTask,
  onDownloadFiles,
  onChainItemClick,
  hideTaskNameColumn,
}: ConnectedProps) {
  const allHistory = useTaskHistory({
    status: statusFilter,
    pollingIntervalMs,
    disablePolling,
    enabled: !taskName,
  });
  const byNameHistory = useTaskHistoryByName(taskName, {
    status: statusFilter,
    pollingIntervalMs,
    disablePolling,
  });
  const stopMutation = useStopTaskHistory();

  const queryResult = taskName ? byNameHistory : allHistory;
  const rows: TaskHistoryEntry[] = queryResult.data?.items ?? [];
  const isLoading = queryResult.isLoading;

  const onConfirmStop = useCallback(
    (entry: TaskHistoryEntry) => {
      if (onStopTask) {
        onStopTask(entry);
        return;
      }
      if (entry.id !== null && entry.id !== undefined) {
        stopMutation.mutate(entry.id);
      }
    },
    [onStopTask, stopMutation]
  );

  const canStop = useCallback(
    (entry: TaskHistoryEntry) =>
      !!onStopTask || (entry.id !== null && entry.id !== undefined),
    [onStopTask]
  );

  return (
    <TaskHistoryTableView
      rows={rows}
      isLoading={isLoading}
      resolveUserName={resolveUserName}
      onViewLogs={onViewLogs}
      onDownloadFiles={onDownloadFiles}
      onChainItemClick={onChainItemClick}
      hideTaskNameColumn={hideTaskNameColumn}
      onConfirmStop={onConfirmStop}
      isStopping={stopMutation.isPending}
      canStop={canStop}
    />
  );
}

interface PresentationalProps extends Omit<
  TaskHistoryTableProps,
  'taskName' | 'statusFilter' | 'pollingIntervalMs' | 'disablePolling'
> {
  data: TaskHistoryEntry[];
}

function PresentationalTaskHistoryTable({
  data,
  isLoading,
  resolveUserName,
  onViewLogs,
  onStopTask,
  isStopping,
  onDownloadFiles,
  onChainItemClick,
  hideTaskNameColumn,
}: PresentationalProps) {
  const onConfirmStop = useCallback(
    (entry: TaskHistoryEntry) => onStopTask?.(entry),
    [onStopTask]
  );

  const canStop = useCallback(() => !!onStopTask, [onStopTask]);

  return (
    <TaskHistoryTableView
      rows={data}
      isLoading={!!isLoading}
      resolveUserName={resolveUserName}
      onViewLogs={onViewLogs}
      onDownloadFiles={onDownloadFiles}
      onChainItemClick={onChainItemClick}
      hideTaskNameColumn={hideTaskNameColumn}
      onConfirmStop={onConfirmStop}
      isStopping={!!isStopping}
      canStop={canStop}
    />
  );
}

/**
 * Render task-history rows.
 *
 * Two modes:
 * - Connected: when `data` is omitted, the component fetches via React Query
 *   (requires a `QueryClientProvider`) and polls while running rows exist.
 * - Presentational: when `data` is provided, the component renders the rows
 *   verbatim with no React Query usage — safe for stories, tests, and any
 *   consumer that already owns the data.
 */
export function TaskHistoryTable(props: TaskHistoryTableProps) {
  if (props.data !== undefined) {
    return <PresentationalTaskHistoryTable {...props} data={props.data} />;
  }
  return <ConnectedTaskHistoryTable {...props} />;
}
