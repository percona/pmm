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
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import AddIcon from '@mui/icons-material/Add';
import ScheduleIcon from '@mui/icons-material/Schedule';
import { useSnackbar } from 'notistack';
import {
  DEFAULT_PLUGIN_LIST_LIMIT,
  DEFAULT_PLUGIN_LIST_OFFSET,
  RUNNING_STATUSES,
  useDeletePluginEntity,
  usePluginEntityList,
  usePluginTasks,
  type PluginSchema,
  type TaskHistoryStatus,
} from '@sep/api';
import {
  SchemaListView,
  type RenderListColumnOverride,
} from '../SchemaListView';
import { DeleteConfirmDialog } from './DeleteConfirmDialog';

interface PluginListPageProps {
  schema: PluginSchema;
  pluginName: string;
  mockTasks?: Record<string, unknown>[];
  mockEntityItems?: Record<string, Record<string, unknown>[]>;
  /** When true, hide the create action and row navigation to detail. */
  listOnly?: boolean;
  /** When true, hide only the create button (row navigation still enabled). */
  hideCreate?: boolean;
  /** When true, hide entity tabs (nodes / services / …). */
  hideEntityTabs?: boolean;
  /**
   * When true, suppress the generic Schedules button. Opt-in for nested mounts
   * (e.g. inventory) that render their own scheduling button with a correct
   * relative target; the default still renders the button.
   */
  hideScheduleButton?: boolean;
  /** When set, overrides ``useParams().entityName`` (nested routes that use a fixed segment like ``nodes``). */
  entityNameOverride?: string;
  /**
   * When set, row clicks navigate to this absolute path (e.g. nested inventory:
   * ``(row) => \`\${pathname}/\${row.id}\```).
   */
  rowClickHref?: (row: Record<string, unknown>) => string;
  /** When true, list views that declare an ``actions`` column show row delete controls. */
  allowListEntityDelete?: boolean;
  /** Optional per-cell override for non-`actions` columns (falls back to `formatCellValue`). */
  renderListColumn?: RenderListColumnOverride;
  /**
   * Disable the poll-while-running task-list refresh. Opt-in escape hatch for
   * tests and stories so they issue no repeat requests, mirroring
   * ``SchemaListView``'s ``disableSchedulePolling``.
   */
  disableTaskPolling?: boolean;
}

export function PluginListPage({
  schema,
  pluginName,
  mockTasks,
  mockEntityItems,
  listOnly = false,
  hideCreate = false,
  hideEntityTabs = false,
  hideScheduleButton = false,
  entityNameOverride,
  rowClickHref,
  allowListEntityDelete = false,
  renderListColumn,
  disableTaskPolling = false,
}: PluginListPageProps) {
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const [pendingDelete, setPendingDelete] = useState<{
    id: string;
    name?: string;
  } | null>(null);
  const { entityName: entityNameParam } = useParams<{ entityName?: string }>();
  const entityName = entityNameOverride ?? entityNameParam;
  const entitySchema = useMemo(
    () => schema.entities?.find((e) => e.name === entityName),
    [schema.entities, entityName]
  );
  const multi = Boolean(schema.entities?.length && entityName && entitySchema);

  const paginationScope = `${pluginName}:${entityName ?? ''}`;
  const [listPage, setListPage] = useState({
    scope: paginationScope,
    offset: DEFAULT_PLUGIN_LIST_OFFSET,
    limit: DEFAULT_PLUGIN_LIST_LIMIT,
  });

  // Derive the active page from scope so a tab change uses offset 0 on the
  // same render (a post-paint useEffect would still fire one stale-offset
  // request and flash an empty page when the new entity has fewer rows).
  const activeListPage =
    listPage.scope === paginationScope
      ? listPage
      : {
          scope: paginationScope,
          offset: DEFAULT_PLUGIN_LIST_OFFSET,
          limit: DEFAULT_PLUGIN_LIST_LIMIT,
        };

  if (listPage.scope !== paginationScope) {
    setListPage(activeListPage);
  }

  const listQueryOptions = {
    offset: activeListPage.offset,
    limit: activeListPage.limit,
  };

  const singleQuery = usePluginTasks(pluginName, mockTasks, {
    enabled: !multi,
    disablePolling: disableTaskPolling,
    ...listQueryOptions,
  });
  const entityQuery = usePluginEntityList(
    pluginName,
    entityName ?? '',
    multi ? mockEntityItems?.[entityName!] : undefined,
    { enabled: multi, ...listQueryOptions }
  );

  const { data: listResult, isLoading } = multi ? entityQuery : singleQuery;
  const rows = listResult?.items ?? [];
  const listPagination = listResult?.pagination ?? null;
  // "Currently running" summarises the task list only. Entity lists are not
  // task lists (they carry no running state and are not polled), so the count
  // stays at zero there and the affordance never renders.
  const runningCount = useMemo(
    () =>
      multi
        ? 0
        : rows.filter((row) =>
            RUNNING_STATUSES.has(row.status as TaskHistoryStatus)
          ).length,
    [multi, rows]
  );
  const listView = multi ? entitySchema!.list_view : schema.list_view!;
  const title = multi ? entitySchema!.display_name : schema.display_name;
  const description = multi ? entitySchema?.description : schema.description;

  const hasActionsColumn = listView.columns.some((c) => c.format === 'actions');
  const deleteEntity = useDeletePluginEntity(
    pluginName,
    entityName ?? '',
    multi ? mockEntityItems?.[entityName!] : undefined
  );

  const onDeleteRow =
    allowListEntityDelete && multi && entityName && hasActionsColumn
      ? (row: Record<string, unknown>) => {
          const rid = row.id;
          if (rid === undefined || rid === null) {
            return;
          }
          const rawName = row.name;
          const rowName =
            typeof rawName === 'string' && rawName.trim()
              ? rawName.trim()
              : typeof rawName === 'number'
                ? String(rawName)
                : undefined;
          setPendingDelete({ id: String(rid), name: rowName });
        }
      : undefined;

  const confirmListDelete = () => {
    const sid = pendingDelete?.id;
    setPendingDelete(null);
    if (!sid) {
      return;
    }
    deleteEntity.mutate(sid, {
      onSuccess: () => {
        enqueueSnackbar(`${title} deleted`, { variant: 'success' });
      },
      onError: (err: Error) => {
        enqueueSnackbar(err.message || 'Delete failed', { variant: 'error' });
      },
    });
  };

  return (
    <Box>
      {multi && schema.entities && !hideEntityTabs && (
        <Tabs
          sx={{ mb: 2 }}
          value={entityName}
          onChange={(_, value) => {
            if (typeof value === 'string' && value !== entityName) {
              navigate(`../${value}`, { relative: 'path' });
            }
          }}
          variant="scrollable"
          scrollButtons="auto"
        >
          {schema.entities.map((e) => (
            <Tab key={e.name} label={e.display_name} value={e.name} />
          ))}
        </Tabs>
      )}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          mb: 3,
          gap: 2,
        }}
      >
        <Box>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Typography variant="h4">{title}</Typography>
            {runningCount > 0 && (
              <Chip
                size="small"
                color="info"
                label={`Currently running (${runningCount})`}
                data-testid="currently-running"
              />
            )}
          </Stack>
          {description && (
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
              {description}
            </Typography>
          )}
        </Box>
        {!listOnly && (
          <Stack direction="row" spacing={1}>
            {!hideScheduleButton && schema.capabilities?.scheduling && (
              <Button
                variant="outlined"
                startIcon={<ScheduleIcon />}
                onClick={() => navigate('schedule', { relative: 'path' })}
                data-testid="plugin-schedule-link"
              >
                Schedules
              </Button>
            )}
            {!hideCreate && (
              <Button
                variant="contained"
                startIcon={<AddIcon />}
                onClick={() => navigate('new', { relative: 'path' })}
              >
                New {multi ? title : schema.display_name}
              </Button>
            )}
          </Stack>
        )}
      </Box>

      <DeleteConfirmDialog
        open={pendingDelete !== null}
        onClose={() => setPendingDelete(null)}
        onConfirm={confirmListDelete}
        title={`Delete from ${schema.display_name}?`}
        description={
          pendingDelete
            ? pendingDelete.name
              ? `Permanently delete ${title} "${pendingDelete.name}" (id ${pendingDelete.id}) from ${schema.display_name}? This cannot be undone.`
              : `Permanently delete ${title} (id ${pendingDelete.id}) from ${schema.display_name}? This cannot be undone.`
            : ''
        }
      />

      <SchemaListView
        listView={listView}
        data={rows}
        isLoading={isLoading}
        pluginName={pluginName}
        pagination={
          listPagination
            ? {
                total: listPagination.total,
                offset: listPagination.offset,
                limit: listPagination.limit,
                onChange: ({ offset, limit }) =>
                  setListPage({ scope: paginationScope, offset, limit }),
              }
            : null
        }
        onRowClick={
          listOnly
            ? undefined
            : multi
              ? (row) =>
                  rowClickHref
                    ? navigate(rowClickHref(row as Record<string, unknown>))
                    : navigate(String(row.id), { relative: 'path' })
              : (row) => {
                  // Backend per-plugin detail/delete routes look up by `task_name`
                  // (string), not numeric `id`. The first listView column is
                  // typically `name`; fall back to id only if name is absent.
                  const key = row.name ?? row.id;
                  if (key !== undefined && key !== null) {
                    navigate(`task/${encodeURIComponent(String(key))}`, {
                      relative: 'path',
                    });
                  }
                }
        }
        onDeleteRow={onDeleteRow}
        deletingRowId={deleteEntity.isPending ? deleteEntity.variables : null}
        renderListColumn={renderListColumn}
      />
    </Box>
  );
}
