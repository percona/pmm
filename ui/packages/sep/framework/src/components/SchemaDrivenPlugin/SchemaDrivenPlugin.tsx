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

import { useMemo, useState, type ReactNode } from 'react';
import {
  Routes,
  Route,
  Navigate,
  useLocation,
  useNavigate,
  useParams,
} from 'react-router-dom';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Typography from '@mui/material/Typography';
import IconButton from '@mui/material/IconButton';
import Skeleton from '@mui/material/Skeleton';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { useSnackbar } from 'notistack';
import {
  useAuth,
  usePluginSchema,
  usePluginEntityDetail,
  useUpdatePluginEntity,
  type PluginEntitySchema,
  type PluginSchema,
} from '@sep/api';
import { ReadOnlyNotice } from '../ReadOnlyNotice';
import {
  SchemaFormRenderer,
  coerceFormValues,
  flattenSectionFields,
} from '../SchemaFormRenderer';
import type { RenderFieldOverride } from '../SchemaFormRenderer/types';
import {
  EMPTY_SUBMIT_ERROR,
  mapSubmitError,
  type SubmitErrorState,
} from './submitErrorMapping';
import { PluginListPage } from './PluginListPage';
import { PluginCreatePage } from './PluginCreatePage';
import { PluginTaskEditPage } from './PluginTaskEditPage';
import { PluginDetailPage, type TaskExecuteAction } from './PluginDetailPage';
import { PluginSchedulePage } from './PluginSchedulePage';
import {
  RelatedAppTabBar,
  resolveRelatedAppActiveSegment,
} from './RelatedAppTabBar';
import { resolvePluginRouteBase } from './routeBase';
import type { RenderFormSlot } from './types';
import type { RenderListColumnOverride } from '../SchemaListView';

interface SchemaDrivenPluginProps {
  pluginName: string;
  /** Absolute list route prefix when the plugin is not mounted under ``/apps/{name}``. */
  routeBase?: string;
  mockSchema?: PluginSchema;
  mockTasks?: Record<string, unknown>[];
  mockEntityItems?: Record<string, Record<string, unknown>[]>;
  /** When true, only the list table is shown (no create, detail, or edit routes). */
  listOnly?: boolean;
  /**
   * Browse mode: list + read-only detail (and optional ``renderEntityDetailChildren``),
   * without create / edit / delete routes on detail chrome. List row delete is separate
   * (see ``allowListEntityDelete``).
   */
  browseOnly?: boolean;
  /** Keys to hide from the generic detail field dump (nested relations, etc.). */
  suppressDetailKeys?: string[];
  /** Replace the default Execute button on single-task detail pages. */
  getTaskExecuteActions?: (
    task: Record<string, unknown>
  ) => TaskExecuteAction[] | undefined;
  /** Task names whose execution history should appear on the Execution History tab. */
  getTaskHistoryNames?: (task: Record<string, unknown>) => string[] | undefined;
  /** Extra overview content on single-task detail pages. */
  renderTaskDetailChildren?: (args: {
    task: Record<string, unknown>;
    pluginName: string;
    schema: PluginSchema;
  }) => ReactNode;
  /** Hide multi-entity tab bar (e.g. inventory uses breadcrumbs instead). */
  hideEntityTabs?: boolean;
  /**
   * When true, entity list tables that declare an ``actions`` column show a per-row delete
   * control (inventory uses this with browse-only detail chrome).
   */
  allowListEntityDelete?: boolean;
  renderEntityDetailChildren?: (args: {
    entityName: string;
    record: Record<string, unknown>;
    schema: PluginSchema;
    pathname: string;
    pluginName: string;
    mockEntityItems?: Record<string, Record<string, unknown>[]>;
    allowListEntityDelete?: boolean;
  }) => ReactNode;
  /**
   * Per-field widget override, threaded to the create and edit forms. Applied
   * after conditional gating; overrides must write through react-hook-form.
   */
  renderField?: RenderFieldOverride;
  /** Whole-form slot for the create page; framework keeps chrome / mutation / snackbars. */
  renderCreateForm?: RenderFormSlot;
  /** Whole-form slot for the edit page; framework keeps chrome / mutation / snackbars. */
  renderEditForm?: RenderFormSlot;
  /** Per-cell list column override (non-`actions` columns), threaded to the list page. */
  renderListColumn?: RenderListColumnOverride;
  /**
   * Disable the poll-while-running task-list refresh on the single-task list
   * page. Threaded to {@link PluginListPage} so stories/tests mounting a real
   * list with a running row issue no repeat requests, mirroring
   * `disableSchedulePolling`.
   */
  disableTaskPolling?: boolean;
}

function PluginEditPage({
  schema,
  pluginName,
  mockEntityItems,
  renderField,
  renderEditForm,
}: {
  schema: PluginSchema;
  pluginName: string;
  mockEntityItems?: Record<string, Record<string, unknown>[]>;
  renderField?: RenderFieldOverride;
  renderEditForm?: RenderFormSlot;
}) {
  const navigate = useNavigate();
  const { canMutate } = useAuth();
  const { entityName, id } = useParams<{ entityName?: string; id: string }>();
  const entitySchema = useMemo(
    () =>
      schema.entities?.find((e: PluginEntitySchema) => e.name === entityName),
    [schema.entities, entityName]
  );
  const multi = Boolean(schema.entities?.length && entityName && entitySchema);

  const { enqueueSnackbar } = useSnackbar();
  const updateEntity = useUpdatePluginEntity(
    pluginName,
    entityName ?? '',
    multi ? mockEntityItems?.[entityName!] : undefined
  );

  const { data: item, isLoading } = usePluginEntityDetail(
    pluginName,
    entityName ?? '',
    id,
    multi ? mockEntityItems?.[entityName!] : undefined,
    { enabled: multi && Boolean(id) }
  );

  const title = entitySchema?.display_name ?? schema.display_name;
  const sections = entitySchema?.forms ?? schema.forms!;

  const [{ submitError, fieldErrors }, setSubmitErrorState] =
    useState<SubmitErrorState>(EMPTY_SUBMIT_ERROR);

  const handleSubmit = (data: Record<string, unknown>) => {
    if (!id || !multi) {
      return;
    }
    // Clear any prior failure so the banner / inline errors reset before the
    // new attempt resolves.
    setSubmitErrorState(EMPTY_SUBMIT_ERROR);
    updateEntity.mutate(
      // Coerce at the submit boundary so whole-form slots that bypass
      // SchemaFormRenderer's internal coercion still send a backend-ready
      // payload. Idempotent for the default (already-coerced) path.
      {
        id,
        values: coerceFormValues(data, flattenSectionFields(sections)),
      },
      {
        onSuccess: () => {
          enqueueSnackbar(`${title} updated`, { variant: 'success' });
          navigate('..', { relative: 'path' });
        },
        onError: (error: unknown) => {
          const message =
            error instanceof Error ? error.message : 'Failed to update';
          // Transient toast is unchanged; 422s additionally map to a persistent
          // banner plus inline per-field errors.
          enqueueSnackbar(message, { variant: 'error' });
          setSubmitErrorState(mapSubmitError(error, sections, message));
        },
      }
    );
  };

  // Editing is a mutation, so the whole page is the control (see
  // PluginCreatePage), and the back chrome stays so a URL arrival is not a
  // dead end.
  if (!canMutate) {
    return (
      <Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 3 }}>
          <IconButton
            onClick={() => navigate('..', { relative: 'path' })}
            aria-label={`Back to ${title}`}
          >
            <ArrowBackIcon />
          </IconButton>
          <Typography variant="h4">
            {title} #{id}
          </Typography>
        </Box>
        <ReadOnlyNotice
          action={`edit ${title}`}
          testId="plugin-entity-edit-read-only"
        />
      </Box>
    );
  }

  if (!multi || !entitySchema) {
    return (
      <Box sx={{ py: 2 }}>
        <Typography color="text.secondary">
          Edit is only available for multi-entity plugins.
        </Typography>
      </Box>
    );
  }

  if (isLoading) {
    return (
      <Box>
        <Skeleton variant="text" width={300} height={40} />
        <Skeleton variant="rectangular" height={200} sx={{ mt: 2 }} />
      </Box>
    );
  }

  if (!item) {
    return (
      <Box sx={{ py: 2 }}>
        <Typography variant="h5">Not found</Typography>
      </Box>
    );
  }

  const defaultValues = item as Record<string, unknown>;

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 3 }}>
        <IconButton onClick={() => navigate('..', { relative: 'path' })}>
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h4">
          Edit {title} #{id}
        </Typography>
      </Box>

      {renderEditForm?.({
        sections,
        onSubmit: handleSubmit,
        loading: updateEntity.isPending,
        defaultValues,
        capabilities: schema.capabilities,
        renderField,
        submitError,
        fieldErrors,
      }) ?? (
        <SchemaFormRenderer
          sections={sections}
          onSubmit={handleSubmit}
          loading={updateEntity.isPending}
          submitLabel={`Save ${title}`}
          submitError={submitError}
          fieldErrors={fieldErrors}
          defaultValues={defaultValues}
          renderField={renderField}
        />
      )}
    </Box>
  );
}

export function SchemaDrivenPlugin({
  pluginName,
  routeBase,
  mockSchema,
  mockTasks,
  mockEntityItems,
  listOnly = false,
  browseOnly = false,
  suppressDetailKeys,
  getTaskExecuteActions,
  getTaskHistoryNames,
  renderTaskDetailChildren,
  hideEntityTabs = false,
  allowListEntityDelete = false,
  renderEntityDetailChildren,
  renderField,
  renderCreateForm,
  renderEditForm,
  renderListColumn,
  disableTaskPolling = false,
}: SchemaDrivenPluginProps) {
  const { pathname } = useLocation();
  const {
    data: schema,
    isLoading,
    error,
  } = usePluginSchema(pluginName, mockSchema);
  const showMutationRoutes = !listOnly && !browseOnly;
  const showDetailRoutes = !listOnly;
  const resolvedRouteBase = resolvePluginRouteBase(
    pluginName,
    routeBase
  ).replace(/\/+$/, '');
  const relatedApps = schema?.related_apps ?? [];
  const hasRelatedApps = relatedApps.length > 0;

  if (isLoading && !schema) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error && !schema) {
    return (
      <Box sx={{ py: 4 }}>
        <Typography color="error">
          Failed to load plugin schema: {error.message}
        </Typography>
      </Box>
    );
  }

  if (!schema) {
    return null;
  }

  if (schema.entities?.length) {
    const first = schema.entities[0].name;
    return (
      <Routes>
        <Route index element={<Navigate to={first} replace />} />
        <Route
          path=":entityName"
          element={
            <PluginListPage
              schema={schema}
              pluginName={pluginName}
              mockEntityItems={mockEntityItems}
              listOnly={listOnly}
              hideCreate={browseOnly && !listOnly}
              hideEntityTabs={hideEntityTabs}
              allowListEntityDelete={allowListEntityDelete}
              renderListColumn={renderListColumn}
            />
          }
        />
        {showMutationRoutes && (
          <>
            <Route
              path=":entityName/new"
              element={
                <PluginCreatePage
                  schema={schema}
                  pluginName={pluginName}
                  mockEntityItems={mockEntityItems}
                  renderField={renderField}
                  renderCreateForm={renderCreateForm}
                />
              }
            />
            <Route
              path=":entityName/:id/edit"
              element={
                <PluginEditPage
                  schema={schema}
                  pluginName={pluginName}
                  mockEntityItems={mockEntityItems}
                  renderField={renderField}
                  renderEditForm={renderEditForm}
                />
              }
            />
          </>
        )}
        {showDetailRoutes && (
          <Route
            path=":entityName/:id"
            element={
              <PluginDetailPage
                schema={schema}
                pluginName={pluginName}
                mockEntityItems={mockEntityItems}
                browseOnly={browseOnly}
                suppressDetailKeys={suppressDetailKeys}
                renderEntityDetailChildren={renderEntityDetailChildren}
                allowListEntityDelete={allowListEntityDelete}
              />
            }
          />
        )}
      </Routes>
    );
  }

  const nestedPluginProps = {
    mockTasks,
    mockEntityItems,
    listOnly,
    browseOnly,
    suppressDetailKeys,
    getTaskExecuteActions,
    getTaskHistoryNames,
    renderTaskDetailChildren,
    hideEntityTabs,
    allowListEntityDelete,
    renderEntityDetailChildren,
    renderField,
    renderCreateForm,
    renderEditForm,
    renderListColumn,
  };

  const singleTaskRoutes = (
    <Routes>
      <Route
        index
        element={
          <PluginListPage
            schema={schema}
            pluginName={pluginName}
            mockTasks={mockTasks}
            mockEntityItems={mockEntityItems}
            listOnly={listOnly}
            hideCreate={browseOnly && !listOnly}
            renderListColumn={renderListColumn}
            disableTaskPolling={disableTaskPolling}
          />
        }
      />
      {showMutationRoutes && (
        <>
          <Route
            path="new"
            element={
              <PluginCreatePage
                schema={schema}
                pluginName={pluginName}
                mockTasks={mockTasks}
                renderField={renderField}
                renderCreateForm={renderCreateForm}
              />
            }
          />
          <Route
            path="task/:id/edit"
            element={
              <PluginTaskEditPage
                schema={schema}
                pluginName={pluginName}
                mockTasks={mockTasks}
                renderField={renderField}
                renderEditForm={renderEditForm}
              />
            }
          />
        </>
      )}
      {/* Schedule route is only registered when the plugin schema opts in;
          otherwise direct navigation to `/apps/<name>/schedule` would
          show the panel even though the entry-point buttons (which gate on
          the same capability) are hidden. */}
      {schema.capabilities?.scheduling && (
        <Route
          path="schedule"
          element={<PluginSchedulePage pluginName={pluginName} />}
        />
      )}
      {showDetailRoutes && (
        <Route
          path="task/:id/*"
          element={
            <PluginDetailPage
              schema={schema}
              pluginName={pluginName}
              routeBase={routeBase}
              mockTasks={mockTasks}
              browseOnly={browseOnly}
              suppressDetailKeys={suppressDetailKeys}
              getTaskExecuteActions={getTaskExecuteActions}
              getTaskHistoryNames={getTaskHistoryNames}
              renderTaskDetailChildren={renderTaskDetailChildren}
              renderEntityDetailChildren={renderEntityDetailChildren}
              allowListEntityDelete={allowListEntityDelete}
            />
          }
        />
      )}
      {hasRelatedApps &&
        relatedApps.map((related) => (
          <Route
            key={related.app_key}
            path={`${related.route_segment}/*`}
            element={
              <SchemaDrivenPlugin
                pluginName={related.app_key}
                routeBase={`${resolvedRouteBase}/${related.route_segment}`}
                {...nestedPluginProps}
              />
            }
          />
        ))}
    </Routes>
  );

  if (hasRelatedApps) {
    return (
      <Box>
        <RelatedAppTabBar
          parentLabel={schema.display_name}
          routeBase={resolvedRouteBase}
          relatedApps={relatedApps}
          activeSegment={resolveRelatedAppActiveSegment(
            pathname,
            resolvedRouteBase,
            relatedApps
          )}
        />
        {singleTaskRoutes}
      </Box>
    );
  }

  return singleTaskRoutes;
}
