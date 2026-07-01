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

import { useMemo, type ReactNode } from 'react';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import { MaterialReactTable, type MRT_ColumnDef } from 'material-react-table';
import type { ListColumn, ListView } from '@sep/api';
import { SEP_TABLE_CLASS } from '../../constants';
import { ScheduleCell } from '../ScheduleCell';
import {
  selectSchedule,
  useScheduledTasksForPlugin,
  type PeriodicTaskResponse,
} from '../ScheduledTasksPanel';

/** Argument bag handed to a {@link RenderListColumnOverride}. */
export interface RenderListColumnArgs {
  /** The matched column's `key`. */
  columnKey: string;
  /** The raw cell value for this column. */
  value: unknown;
  /** The full row record. */
  row: Record<string, unknown>;
}

/**
 * Per-cell list column override for {@link SchemaListView}.
 *
 * Called for every non-`actions` cell. Return custom UI to render that cell, or
 * return `undefined` to fall back to the framework's `formatCellValue`. Keying
 * is the override's own concern: branch on `args.columnKey` and return
 * `undefined` for columns it does not handle. The `actions` column (bespoke
 * row-delete control) is never routed through this override.
 */
export type RenderListColumnOverride = (
  args: RenderListColumnArgs
) => ReactNode;

interface SchemaListViewProps {
  listView: ListView;
  data: Record<string, unknown>[];
  isLoading?: boolean;
  /**
   * Owning plugin name. Required for `schedule`-format columns: the schedule
   * fetch and the client-side task-name join only run when this is set and the
   * list view declares a schedule column, so plugins without one issue no
   * periodic-tasks request.
   */
  pluginName?: string;
  /** Disable schedule-list polling. Used by stories/tests. */
  disableSchedulePolling?: boolean;
  onRowClick?: (row: Record<string, unknown>) => void;
  /** When set, ``format: 'actions'`` columns render a delete control for that row. */
  onDeleteRow?: (row: Record<string, unknown>) => void;
  /** Row id currently being deleted (disables that row's button). */
  deletingRowId?: string | null;
  /**
   * Optional per-cell override for non-`actions` columns. Returns custom UI for
   * a cell or `undefined` to fall back to `formatCellValue`. See
   * {@link RenderListColumnOverride}.
   */
  renderListColumn?: RenderListColumnOverride;
}

function formatCellValue(
  value: unknown,
  format: ListColumn['format']
): ReactNode {
  if (value === null) {
    return '—';
  }
  const str = String(value);

  switch (format) {
    case 'chip':
      return <Chip label={str} size="small" />;
    case 'status':
      return (
        <Chip
          label={str}
          size="small"
          color={
            str === 'completed' || str === 'success'
              ? 'success'
              : str === 'failed' || str === 'error'
                ? 'error'
                : str === 'running' || str === 'in_progress'
                  ? 'info'
                  : 'default'
          }
        />
      );
    case 'date':
      return new Date(str).toLocaleDateString();
    case 'relative': {
      const diff = Date.now() - new Date(str).getTime();
      const mins = Math.floor(diff / 60000);
      if (mins < 1) {
        return 'just now';
      }
      if (mins < 60) {
        return `${mins}m ago`;
      }
      const hours = Math.floor(mins / 60);
      if (hours < 24) {
        return `${hours}h ago`;
      }
      const days = Math.floor(hours / 24);
      return `${days}d ago`;
    }
    case 'code':
      return (
        <Typography
          variant="body2"
          sx={{ fontFamily: "'Roboto Mono', monospace" }}
        >
          {str}
        </Typography>
      );
    default:
      return str;
  }
}

/** Stable empty lookup so plugins without a schedule column never re-key. */
const EMPTY_SCHEDULE = new Map<string, PeriodicTaskResponse>();

/**
 * SchemaListView entry point.
 *
 * The schedule fetch and the client-side task-name join only happen when the
 * list view declares a `schedule`-format column and a plugin name is provided.
 * In that case rendering is delegated to {@link ScheduleJoinedListView}, which
 * is the only place that mounts `useScheduledTasksForPlugin` — so plugins
 * without a schedule column issue no periodic-tasks request (and need no
 * QueryClient).
 */
export function SchemaListView(props: SchemaListViewProps) {
  const { listView, pluginName } = props;
  const hasScheduleColumn = listView.columns.some(
    (c) => c.format === 'schedule'
  );
  if (hasScheduleColumn && pluginName) {
    return <ScheduleJoinedListView {...props} pluginName={pluginName} />;
  }
  return <SchemaListViewCore {...props} scheduleByTask={EMPTY_SCHEDULE} />;
}

/**
 * Fetches the plugin's periodic tasks, builds the by-name lookup, and renders
 * the table. Isolated from {@link SchemaListView} so the schedule hook is only
 * mounted when a schedule column is actually present.
 */
function ScheduleJoinedListView(
  props: SchemaListViewProps & { pluginName: string }
) {
  const { pluginName, disableSchedulePolling = false } = props;
  const { periodicTasks, isLoading: scheduleLoading } =
    useScheduledTasksForPlugin(pluginName, {
      disablePolling: disableSchedulePolling,
    });

  const scheduleByTask = useMemo(() => {
    // A task may own several periodic schedules; group by task name and pick a
    // single one with the same rule the detail summary uses, so the list cell
    // and the summary never disagree.
    const grouped = new Map<string, PeriodicTaskResponse[]>();
    for (const task of periodicTasks) {
      const existing = grouped.get(task.task);
      if (existing) {
        existing.push(task);
      } else {
        grouped.set(task.task, [task]);
      }
    }
    const selected = new Map<string, PeriodicTaskResponse>();
    for (const [name, candidates] of grouped) {
      const pick = selectSchedule(candidates);
      if (pick) {
        selected.set(name, pick);
      }
    }
    return selected;
  }, [periodicTasks]);

  return (
    <SchemaListViewCore
      {...props}
      scheduleByTask={scheduleByTask}
      scheduleLoading={scheduleLoading}
    />
  );
}

function SchemaListViewCore({
  listView,
  data,
  isLoading = false,
  onRowClick,
  onDeleteRow,
  deletingRowId,
  renderListColumn,
  scheduleByTask,
  scheduleLoading = false,
}: SchemaListViewProps & {
  scheduleByTask: Map<string, PeriodicTaskResponse>;
  scheduleLoading?: boolean;
}) {
  const columns = useMemo<MRT_ColumnDef<Record<string, unknown>>[]>(
    () =>
      listView.columns.map((col) => {
        if (col.format === 'schedule') {
          return {
            id: col.key,
            accessorKey: col.key,
            header: col.label,
            enableSorting: col.sortable ?? false,
            Cell: ({ row }) => {
              const name = row.original.name;
              // Trim to match how the detail summary derives its lookup key
              // (PluginDetailPage trims `task.name`), so the list cell and the
              // summary join to the same schedule even with stray whitespace.
              const matched =
                name === undefined || name === null
                  ? undefined
                  : scheduleByTask.get(String(name).trim());
              return (
                <ScheduleCell task={matched} isLoading={scheduleLoading} />
              );
            },
          };
        }
        if (col.format === 'actions') {
          return {
            id: col.key,
            accessorKey: col.key,
            header: col.label,
            enableSorting: false,
            size: 72,
            Cell: ({ row }) => {
              const id = row.original.id;
              if (id === undefined || id === null || !onDeleteRow) {
                return null;
              }
              const sid = String(id);
              return (
                <IconButton
                  size="small"
                  color="error"
                  aria-label="Delete"
                  disabled={deletingRowId === sid}
                  onClick={(e) => {
                    e.stopPropagation();
                    onDeleteRow(row.original);
                  }}
                >
                  <DeleteOutlineIcon fontSize="small" />
                </IconButton>
              );
            },
          };
        }
        return {
          accessorKey: col.key,
          header: col.label,
          enableSorting: col.sortable ?? true,
          Cell: ({ cell, row }) => {
            const value = cell.getValue();
            const overridden = renderListColumn?.({
              columnKey: col.key,
              value,
              row: row.original,
            });
            // `undefined` is the "no override" sentinel (override absent, or it
            // declined this column); `null` is honored so an override can
            // intentionally render an empty cell.
            return overridden === undefined
              ? formatCellValue(value, col.format)
              : overridden;
          },
        };
      }),
    [
      deletingRowId,
      listView.columns,
      onDeleteRow,
      renderListColumn,
      scheduleByTask,
      scheduleLoading,
    ]
  );

  return (
    <MaterialReactTable
      columns={columns}
      data={data}
      state={{ isLoading }}
      enableColumnActions={false}
      enableDensityToggle={false}
      enableFullScreenToggle={false}
      initialState={{
        sorting: listView.default_sort
          ? [
              {
                id: listView.default_sort.replace(/^-/, ''),
                desc: listView.default_sort.startsWith('-'),
              },
            ]
          : [],
        density: 'compact',
      }}
      muiTablePaperProps={{
        className: SEP_TABLE_CLASS,
        elevation: 0,
        variant: 'outlined',
        // The Percona theme's `background.paper` doesn't always resolve to
        // an opaque colour, leaving the table looking transparent against
        // tinted page backgrounds. Force `common.white` (light mode) so the
        // table is always readable; revisit when dark mode lands.
        sx: { bgcolor: 'common.white' },
      }}
      muiTableContainerProps={{
        sx: { bgcolor: 'common.white' },
      }}
      muiTableBodyRowProps={
        onRowClick
          ? ({ row }) => ({
              onClick: () => onRowClick(row.original),
              sx: { cursor: 'pointer' },
            })
          : undefined
      }
    />
  );
}
