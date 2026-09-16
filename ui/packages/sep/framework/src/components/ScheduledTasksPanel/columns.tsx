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

import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import type { MRT_ColumnDef } from 'material-react-table';
import { LastRunStatus } from './LastRunStatus';
import { describePeriod } from './periods';
import { scheduleTimezone } from './timezones';
import { RunTime } from '../TaskRunDetailDrawer/RunTime';
import type { PeriodicTaskResponse } from './hooks';

export interface ScheduleColumnOptions {
  /** Display noun for the first column (for example, `Backup`). */
  itemLabel?: string;
  /**
   * Render the Chain column. Dropped while nothing is chained (PMM-15454):
   * these apps expose a single chainable task, so no schedule can carry a
   * chain and the column was an unbroken run of em dashes advertising
   * something the reader could not use. It returns on its own the moment any
   * schedule carries a chain.
   */
  showChain: boolean;
  /**
   * Render the enable toggle and the edit / delete controls. Dropped for a
   * session that may not mutate, leaving the schedule readable.
   */
  canMutate: boolean;
  /** Disables the enable toggle while an update is in flight. */
  toggling?: boolean;
  onToggleEnabled: (task: PeriodicTaskResponse, nextEnabled: boolean) => void;
  /**
   * Open the edit page for one schedule. Omit to leave the row with delete
   * alone — for a host that has nowhere to send the reader.
   */
  onEdit?: (task: PeriodicTaskResponse) => void;
  onDelete: (task: PeriodicTaskResponse) => void;
  /**
   * Open the execution detail for a schedule's last run. Omit to leave the
   * last-run cell inert.
   *
   * Carries `last_run_at` as well as the name: the periodic-task API reports
   * no `task_history_id`, so the timestamp is the only thing that identifies
   * which run the row is describing.
   */
  onOpenLastRun?: (taskName: string, lastRunAt: string | null) => void;
}

/**
 * The schedules table's columns, in render order.
 *
 * Cell content that used to live in a hand-written `<TableRow>` component; the
 * schedules list is a data table like the other two lists in the app now
 * (PMM-15456), so the rows are the table's to render.
 */
export function scheduleColumns({
  itemLabel = 'Task',
  showChain,
  canMutate,
  toggling,
  onToggleEnabled,
  onEdit,
  onDelete,
  onOpenLastRun,
}: ScheduleColumnOptions): MRT_ColumnDef<PeriodicTaskResponse>[] {
  const columns: MRT_ColumnDef<PeriodicTaskResponse>[] = [
    {
      accessorKey: 'task',
      header: itemLabel,
      size: 180,
    },
    {
      id: 'period',
      header: 'Period',
      accessorFn: (task) => describePeriod(task).display,
      size: 180,
      // Not sortable: the cell is a sentence (`every 10 minutes`, `0 6 * * *`),
      // and sorting sentences puts `every 10 minutes` before `every 2 minutes`
      // while giving cron and interval schedules no shared order at all. The
      // list was not sortable at all before it became a data table, so nothing
      // is lost by declining here rather than inventing a comparator.
      enableSorting: false,
      Cell: ({ row }) => {
        const period = describePeriod(row.original);
        // The zone the schedule actually fires in, which is not the zone its
        // timestamps are rendered in — the panel names that one once for the
        // whole table.
        const firesIn = scheduleTimezone(row.original);
        return (
          <Stack spacing={0.25} alignItems="flex-start">
            {period.tooltip ? (
              <Tooltip title={period.tooltip}>
                <span>{period.display}</span>
              </Tooltip>
            ) : (
              <span>{period.display}</span>
            )}
            <Typography
              variant="caption"
              color="text.secondary"
              data-testid={`scheduled-task-timezone-${row.original.id}`}
            >
              Runs in {firesIn}
            </Typography>
          </Stack>
        );
      },
    },
    {
      accessorKey: 'start_time',
      header: 'Start Time',
      size: 150,
      Cell: ({ row }) => <RunTime value={row.original.start_time} />,
    },
    {
      id: 'last_run',
      header: 'Last Run',
      accessorKey: 'last_run_at',
      size: 170,
      Cell: ({ row }) => (
        <Stack spacing={0.5} alignItems="flex-start">
          <LastRunStatus
            status={row.original.last_run_status}
            lastRunAt={row.original.last_run_at}
            onOpenRun={
              onOpenLastRun
                ? () =>
                    onOpenLastRun(row.original.task, row.original.last_run_at)
                : undefined
            }
          />
          {row.original.last_run_at && (
            <Typography variant="body2" color="text.secondary">
              <RunTime value={row.original.last_run_at} />
            </Typography>
          )}
        </Stack>
      ),
    },
    {
      accessorKey: 'next_run_at',
      header: 'Next Run',
      size: 150,
      Cell: ({ row }) => <RunTime value={row.original.next_run_at} />,
    },
    {
      accessorKey: 'total_run_count',
      header: 'Runs',
      size: 80,
    },
  ];

  if (showChain) {
    columns.push({
      id: 'chain',
      header: 'Chain',
      size: 160,
      // Prose, like Period above.
      enableSorting: false,
      accessorFn: (task) =>
        (task.execute_request?.chain_task_names ?? []).join(' → '),
      Cell: ({ row }) => {
        const chainNames = row.original.execute_request?.chain_task_names ?? [];
        return chainNames.length > 0 ? (
          <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
            {chainNames.join(' → ')}
          </Typography>
        ) : (
          '—'
        );
      },
    });
  }

  columns.push({
    accessorKey: 'enabled',
    header: 'Enabled',
    size: 110,
    enableSorting: false,
    Cell: ({ row }) =>
      canMutate ? (
        <Switch
          checked={row.original.enabled}
          disabled={toggling}
          onChange={(_, checked) => onToggleEnabled(row.original, checked)}
          slotProps={{
            input: { 'aria-label': `Enable ${row.original.task}` },
          }}
        />
      ) : (
        <Typography variant="body2">
          {row.original.enabled ? 'Enabled' : 'Disabled'}
        </Typography>
      ),
  });

  if (canMutate) {
    columns.push({
      id: 'actions',
      header: 'Actions',
      size: 110,
      enableSorting: false,
      Cell: ({ row }) => (
        <Stack direction="row" spacing={0.5}>
          {onEdit && (
            <Tooltip title="Edit">
              <IconButton
                size="small"
                onClick={() => onEdit(row.original)}
                aria-label={`Edit ${row.original.task}`}
                data-testid={`scheduled-task-edit-${row.original.id}`}
              >
                <EditOutlinedIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
          <Tooltip title="Delete">
            <IconButton
              size="small"
              onClick={() => onDelete(row.original)}
              aria-label={`Delete ${row.original.task}`}
              data-testid={`scheduled-task-delete-${row.original.id}`}
            >
              <DeleteOutlineIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Stack>
      ),
    });
  }

  return columns;
}
