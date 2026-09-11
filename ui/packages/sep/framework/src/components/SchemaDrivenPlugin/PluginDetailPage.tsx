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

import {
  lazy,
  Suspense,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import {
  Routes,
  Route,
  useParams,
  useNavigate,
  useLocation,
  Link,
} from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import ScheduleIcon from '@mui/icons-material/Schedule';
import { useSnackbar } from 'notistack';
import {
  useAuth,
  useDeletePluginEntity,
  useDeletePluginTask,
  usePluginEntityDetail,
  usePluginTask,
  usePluginTasks,
  type DetailSection,
  type ListView,
  type PluginEntitySchema,
  type PluginSchema,
  type SepComponents,
} from '@sep/api';
import { resolvePath } from '../../utils/resolvePath';
import { applyValueLabel } from '../../utils/valueLabels';
import { ActionErrorAlert, useActionError } from '../ActionErrorAlert';
import {
  TaskHistoryTable,
  TaskHistoryStatusBadge,
  isTaskHistoryStatus,
  type TaskHistoryEntry,
} from '../TaskHistoryTable';
import {
  LastRunCard,
  resolveOpenedRun,
  TaskRunDetailDrawer,
} from '../TaskRunDetailDrawer';
import { ScheduleSummary } from '../ScheduleSummary';
import { ChainBuilder, type ChainValue } from '../ChainBuilder';
import {
  useExecuteTask,
  useStopTaskHistory,
  useTaskHistoryByNames,
  type TaskExecuteBody,
} from '../../hooks';
import { DeleteConfirmDialog } from './DeleteConfirmDialog';
import {
  detailSyntaxBlockSx,
  type DetailSyntaxLanguage,
} from './detailSyntaxStyles';
import { resolvePluginRouteBase } from './routeBase';
import { getStoredForm } from './storedForm';
import {
  selectConfiguredSettings,
  type ConfiguredSection,
} from './taskConfiguration';
import { StatsCard } from './StatsCard';

const DetailSyntaxHighlighter = lazy(() => import('./DetailSyntaxHighlighter'));

export type { TaskExecuteBody };

/** One executable target on a plugin task detail action bar. */
export interface TaskExecuteAction {
  label: string;
  taskName: string;
  testId?: string;
  confirmMessage?: string;
  executeBody?: TaskExecuteBody;
}

export interface PluginDetailPageProps {
  schema: PluginSchema;
  pluginName: string;
  /** Absolute list route prefix when the plugin is not mounted under ``/apps/{name}``. */
  routeBase?: string;
  mockTasks?: Record<string, unknown>[];
  mockEntityItems?: Record<string, Record<string, unknown>[]>;
  /** Hide edit/delete (browse-only mode for multi-entity plugins). */
  browseOnly?: boolean;
  /** Omit these keys from the auto-rendered detail section (e.g. nested relations shown elsewhere). */
  suppressDetailKeys?: string[];
  /** Replace the default single Execute button with plugin-specific execute targets. */
  getTaskExecuteActions?: (
    task: Record<string, unknown>
  ) => TaskExecuteAction[] | undefined;
  /** Task names whose execution history should appear on the Execution History tab. */
  getTaskHistoryNames?: (task: Record<string, unknown>) => string[] | undefined;
  /** Extra content below the overview cards on single-task detail pages. */
  renderTaskDetailChildren?: (args: {
    task: Record<string, unknown>;
    pluginName: string;
    schema: PluginSchema;
  }) => ReactNode;
  /** Extra content under the main detail card (e.g. child entity tables). */
  renderEntityDetailChildren?: (args: {
    entityName: string;
    record: Record<string, unknown>;
    schema: PluginSchema;
    pathname: string;
    pluginName: string;
    mockEntityItems?: Record<string, Record<string, unknown>[]>;
    allowListEntityDelete?: boolean;
  }) => ReactNode;
  /** Nested inventory: resolve ``entityName`` / ``id`` from named params (e.g. ``nodeId``). */
  detailEntityName?: string;
  detailIdParam?: string;
  /** Optional callback that resolves a custom back/delete target from current pathname. */
  resolveParentPath?: (pathname: string) => string | null;
  /** When true, omit the header row (back button, ``{title} #{id}``, status chip, edit/delete). */
  hideDetailChrome?: boolean;
  /** When true, nested list tables may show row delete (inventory ``actions`` column). */
  allowListEntityDelete?: boolean;
}

/** Screen title when the back/status row is hidden (browse-only multi-entity plugins). */
function detailScreenHeading(
  entityName: string | undefined,
  entityDisplayName: string | undefined
): string | null {
  if (!entityName) {
    return null;
  }
  return `${entityDisplayName ?? entityName} detail`;
}

/** List URL for the current entity tab (path segment before ``:id``), e.g. ``/inventory/services``. */
export function pathToEntityList(pathname: string, entityName: string): string {
  const parts = pathname.split('/').filter(Boolean);
  const idx = parts.lastIndexOf(entityName);
  if (idx < 0) {
    return '/';
  }
  return `/${parts.slice(0, idx + 1).join('/')}`;
}

/** Compact label/value cell size: 1 → 2 → 3 columns. Three columns only at ``lg``
 * so mid-width layouts (sidebar + ~900px viewport) stay readable for long values. */
const DETAIL_FIELD_GRID_SIZE = { xs: 12, sm: 6, lg: 4 } as const;

/** Keep long unbroken strings (emails, hosts) inside their grid cell. */
const detailFieldValueSx = {
  overflowWrap: 'anywhere',
  wordBreak: 'break-word',
} as const;

/** Suspense placeholder sized to match a rendered syntax block. */
function SyntaxBlockFallback() {
  return (
    <Skeleton
      variant="rectangular"
      height={120}
      sx={{ ...detailSyntaxBlockSx, mt: 0.5 }}
    />
  );
}

/**
 * Lazy JSON highlight of an object value. Shared by the detail-field components
 * so the un-hinted object branch renders identically wherever it appears.
 */
function JsonObjectPreview({ value }: { value: unknown }) {
  return (
    <Suspense fallback={<SyntaxBlockFallback />}>
      <DetailSyntaxHighlighter value={value} language="json" />
    </Suspense>
  );
}

/** Compact label/value cell; wide content (JSON / syntax blocks) spans the full row. */
function EntityDetailField({
  label,
  value: rawValue,
  highlightLanguage,
  valueLabels,
}: {
  label: string;
  value: unknown;
  highlightLanguage?: DetailSyntaxLanguage;
  valueLabels?: Record<string, string>;
}) {
  const value = applyValueLabel(rawValue, valueLabels);
  if (value === null || value === undefined || value === '') {
    return null;
  }

  const isWide = typeof value === 'object';

  if (highlightLanguage) {
    return (
      <Grid size={12} sx={{ minWidth: 0 }}>
        <Typography variant="caption" color="text.secondary">
          {label}
        </Typography>
        <Suspense fallback={<SyntaxBlockFallback />}>
          <DetailSyntaxHighlighter value={value} language={highlightLanguage} />
        </Suspense>
      </Grid>
    );
  }

  let display: React.ReactNode;
  if (typeof value === 'boolean') {
    display = value ? 'Yes' : 'No';
  } else if (typeof value === 'object') {
    display = <JsonObjectPreview value={value} />;
  } else {
    display = String(value);
  }

  return (
    <Grid size={isWide ? 12 : DETAIL_FIELD_GRID_SIZE} sx={{ minWidth: 0 }}>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      {typeof value === 'object' ? (
        display
      ) : (
        <Typography variant="body1" sx={detailFieldValueSx}>
          {display}
        </Typography>
      )}
    </Grid>
  );
}

// Framework baseline: numeric `id`, internal worker plumbing (`backend`, `protected`), and
// timestamps already shown in list_view columns. The `data` payload is rendered via
// schema-declared ``detail_view`` sections. Plugin schemas can extend this via
// ``list_view.overview_hidden_fields``. The PII fields are handled by the dedicated
// "PII Anonymization" capability-gated section.
const BASELINE_OVERVIEW_HIDDEN_FIELDS = [
  'id',
  'backend',
  'protected',
  'data',
  'updated_at',
  'last_updated_by',
  'connectivity_warning',
  'anonymize_mask',
  'anonymized_entities',
] as const;

/** Stable empty column set so a schema without a `list_view` never re-memoizes. */
const EMPTY_LIST_COLUMNS: ListView['columns'] = [];

function formatLabel(key: string): string {
  return key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

function SectionCard({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
      <Typography variant="h6" sx={{ mb: 1.5 }}>
        {title}
      </Typography>
      {children}
    </Paper>
  );
}

export function resolveTabFromSplat(
  splat: string | undefined
): 'overview' | 'logs' {
  return splat?.replace(/\/+$/, '').startsWith('logs') ? 'logs' : 'overview';
}

function TaskOverviewDetailField({
  label,
  value: rawValue,
  valueLabels,
}: {
  label: string;
  value: unknown;
  valueLabels?: Record<string, string>;
}) {
  const value = applyValueLabel(rawValue, valueLabels);
  if (value === null || value === undefined || value === '') {
    return null;
  }

  const isWide = typeof value === 'object';
  let display: React.ReactNode;
  if (typeof value === 'boolean') {
    display = value ? 'Yes' : 'No';
  } else if (typeof value === 'object') {
    // No schema highlight hint here — this component is fed from
    // list_view.columns / extra task keys, not DetailField.
    display = <JsonObjectPreview value={value} />;
  } else {
    display = String(value);
  }

  return (
    <Grid size={isWide ? 12 : DETAIL_FIELD_GRID_SIZE} sx={{ minWidth: 0 }}>
      <Typography
        variant="caption"
        color="text.secondary"
        sx={{ display: 'block', mb: 0.5 }}
      >
        {label}
      </Typography>
      {typeof value === 'object' ? (
        display
      ) : (
        <Typography variant="body1" sx={detailFieldValueSx}>
          {display}
        </Typography>
      )}
    </Grid>
  );
}

interface OverviewTabProps {
  schema: PluginSchema;
  task: Record<string, unknown>;
  hiddenFields?: string[];
  /** Owning plugin name; enables the generic schedule summary. */
  pluginName?: string;
  /** Route to the plugin's Schedules screen (the add-a-schedule target). */
  scheduleHref?: string;
  children?: ReactNode;
}

/**
 * The settings this task configured, under the create form's own labels.
 *
 * Rendered from the stored create-form body rather than from the generated
 * config document: see {@link selectConfiguredSettings}. Renders nothing at all
 * when the task left everything at its defaults — an empty card would claim
 * something was configured — and the caller keeps the raw document reachable
 * either way.
 */
function TaskConfigurationCard({
  configured,
}: {
  configured: ConfiguredSection[];
}) {
  return (
    <>
      {configured.map((section) => (
        <SectionCard key={section.title} title={section.title}>
          <Grid container spacing={2}>
            {section.settings.map((setting) => (
              <TaskOverviewDetailField
                key={setting.name}
                label={setting.label}
                // A multi-choice value is a list of stored members; label each
                // and join, rather than handing an array to the JSON preview
                // that the object branch would otherwise render it with.
                value={
                  Array.isArray(setting.value)
                    ? setting.value
                        .map((item) =>
                          String(applyValueLabel(item, setting.valueLabels))
                        )
                        .join(', ')
                    : setting.value
                }
                valueLabels={
                  Array.isArray(setting.value) ? undefined : setting.valueLabels
                }
              />
            ))}
          </Grid>
        </SectionCard>
      ))}
    </>
  );
}

/**
 * Holds the schema's own detail sections, collapsed once something better sits
 * above them.
 *
 * `collapsed` is the decision, not the state: when it is false the children
 * render bare, exactly as they did before this wrapper existed, so a plugin
 * with no stored form sees no disclosure at all rather than one it must open to
 * read the only configuration the page has.
 */
function RawDetailDisclosure({
  collapsed,
  children,
}: {
  collapsed: boolean;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);

  if (!collapsed) {
    return <>{children}</>;
  }

  return (
    <Box sx={{ mb: 2 }}>
      <Button
        size="small"
        variant="text"
        onClick={() => setOpen((prev) => !prev)}
        aria-expanded={open}
        data-testid="raw-configuration-toggle"
      >
        {open ? 'Hide raw configuration' : 'Show raw configuration'}
      </Button>
      {/* Mounted only while open: the config blocks lazy-load a syntax
          highlighter, and a collapsed-but-mounted block would pull that chunk
          on every detail page for something nobody asked to see. */}
      {open && <Box sx={{ mt: 1 }}>{children}</Box>}
    </Box>
  );
}

function DetailViewSectionCard({
  section,
  task,
}: {
  section: DetailSection;
  task: Record<string, unknown>;
}) {
  // Match EntityDetailField's own empty-value rule (undefined / null / '')
  // so a section hides entirely when every field would have rendered blank,
  // and so 0 / false still render as legitimate values.
  const resolved = section.fields
    .map((field) => ({ field, value: resolvePath(task, field.path) }))
    .filter(
      ({ value }) => value !== undefined && value !== null && value !== ''
    );
  if (resolved.length === 0) {
    return null;
  }
  return (
    <SectionCard title={section.title}>
      <Grid container spacing={2}>
        {resolved.map(({ field, value }, idx) => (
          <EntityDetailField
            key={`${field.path}:${field.label}:${idx}`}
            label={field.label}
            value={value}
            highlightLanguage={field.highlight}
            valueLabels={field.value_labels}
          />
        ))}
      </Grid>
    </SectionCard>
  );
}

/**
 * Render the post-creation connectivity-check warning, with an optional link
 * to the run-script log when the check produced a task history. The link is
 * omitted when no `task_history_id` is present (e.g. the Tasks API was
 * unreachable, so no task ran).
 */
function ConnectivityWarningAlert({
  warning,
}: {
  warning: SepComponents['schemas']['framework__ConnectivityWarning'];
}) {
  const [logOpen, setLogOpen] = useState(false);
  const message =
    warning.message || 'Connectivity check returned a warning for this task.';
  const taskHistoryId = warning.task_history_id ?? null;

  return (
    <>
      <Alert
        severity="warning"
        sx={{ mb: 3, whiteSpace: 'pre-wrap' }}
        action={
          taskHistoryId !== null ? (
            // An outlined button, not the inherit-coloured text button this
            // replaces: on the warning Alert's tinted ground that rendered as
            // low-contrast body text rather than something to click.
            <Button
              color="inherit"
              size="small"
              variant="outlined"
              data-testid="connectivity-log-button"
              onClick={() => setLogOpen(true)}
            >
              View log
            </Button>
          ) : undefined
        }
      >
        {message}
      </Alert>
      {taskHistoryId !== null && (
        <TaskRunDetailDrawer
          open={logOpen}
          onClose={() => setLogOpen(false)}
          taskHistoryId={taskHistoryId}
          taskLabel="Connectivity check"
        />
      )}
    </>
  );
}

function OverviewTab({
  schema,
  task,
  hiddenFields = [],
  pluginName,
  scheduleHref,
  children,
}: OverviewTabProps) {
  // The detail/list response model omits `connectivity_warning`; it rides only
  // the create response. PluginCreatePage carries it here via navigation state
  // so the warning surfaces once after a failing post-create check.
  const location = useLocation();
  const navState = (location.state ?? null) as {
    connectivityWarning?: unknown;
  } | null;
  const connectivityWarning =
    task.connectivity_warning ?? navState?.connectivityWarning;
  const [lastRunOpen, setLastRunOpen] = useState(false);
  const taskName =
    typeof task.name === 'string' && task.name.trim()
      ? task.name.trim()
      : undefined;
  // `list_view` is optional: an entity schema reached through an unresolved
  // detail route has no top-level list view, so fall back to the task's own
  // fields (rendered below as `extraEntries`) rather than crashing.
  const columns = schema.list_view?.columns ?? EMPTY_LIST_COLUMNS;
  const schemaHiddenFields = schema.list_view?.overview_hidden_fields;

  const suppressedFields = useMemo(() => {
    // Fields the single-task detail header already surfaces authoritatively:
    // `name` (the h4 task title) and `status` (the header chip). Each is only
    // suppressed under the exact condition the header renders it: the h4 shows
    // `task.name` only when it's a string (otherwise it falls back to the route
    // id), and the chip renders only for a string status. Matching those guards
    // keeps a non-string value from being dropped from both the header and the
    // card. They are the single source of truth for those values, so we
    // de-duplicate them out of the Task information card rather than repeating
    // them below the header.
    const headerShownFields: string[] = [];
    if (typeof task.name === 'string') {
      headerShownFields.push('name');
    }
    if (typeof task.status === 'string') {
      headerShownFields.push('status');
    }
    return new Set([
      ...BASELINE_OVERVIEW_HIDDEN_FIELDS,
      ...headerShownFields,
      ...(schemaHiddenFields ?? []),
      ...hiddenFields,
    ]);
  }, [task.name, task.status, schemaHiddenFields, hiddenFields]);

  // `list_view.columns` targets the table view; in the detail Overview we drop
  // any suppressed column (header/status-chip fields, baseline noise, or
  // backend-curated `overview_hidden_fields`) so no field renders twice.
  const visibleColumns = useMemo(
    () => columns.filter((col) => !suppressedFields.has(col.key)),
    [columns, suppressedFields]
  );

  // Extra fields beyond the schema's list_view columns, excluding internal
  // noise. Lets future plugin schemas surface fields without listing them
  // in `list_view.columns` (which is meant for the table view). Matched against
  // the full column set so a suppressed column can't reappear here as an extra.
  const extraEntries = Object.entries(task).filter(
    ([key]) => !columns.some((c) => c.key === key) && !suppressedFields.has(key)
  );

  // The create-form body the backend stamps on the task, when it has one. A
  // legacy task (or one made through a still-live legacy form) carries none, so
  // the configuration card is skipped and the raw document stays expanded.
  const storedForm = getStoredForm(task);

  // Entity schemas keep their form on the entity, not on the plugin; this
  // Overview only renders for the single-task shape, so the top-level forms are
  // the right source and their absence means there is nothing to join against.
  const configurationSections = schema.forms;

  // Whatever the header and the Task information card already put on screen.
  // A setting is worth listing once; repeating the execution host three inches
  // below the row that states it is noise, not confirmation.
  // Split the schema's declared detail sections by what they hold. A section
  // whose every field carries a syntax `highlight` is a rendered document — the
  // emitted config — and is the only kind this page may demote. Sections of
  // ordinary labelled fields (a restore's destination host and port, say) are
  // primary content the app chose to declare, and stay where the schema put
  // them: the framework does not get to decide an app's fields are secondary
  // just because a stored form exists.
  //
  // `highlight` is an imperfect proxy for "generated": it says how to render a
  // value, not whether the app considers it primary, so an app whose detail
  // view is deliberately one highlighted block would be demoted against its
  // author's intent. Demotion needs a non-empty configuration summary as well
  // (below), which narrows that to an app that has one AND wants its single
  // document kept first. The clean fix is for the section to declare intent —
  // a `DetailSection.generated` flag set by the same backend code that decides
  // `highlight`, defaulting to false so no existing schema changes behaviour —
  // which is a side-car change rather than one this side can make alone.
  const [plainDetailSections, rawDetailSections] = useMemo(() => {
    const sections = schema.detail_view?.sections ?? [];
    const plain: DetailSection[] = [];
    const raw: DetailSection[] = [];
    for (const section of sections) {
      const isRaw =
        section.fields.length > 0 &&
        section.fields.every((field) => Boolean(field.highlight));
      (isRaw ? raw : plain).push(section);
    }
    return [plain, raw];
  }, [schema.detail_view]);

  const configurationExcludedNames = useMemo(
    () => {
      // Exactly what the Task information card puts on screen: its visible
      // columns plus the extra record keys, which is `columns ∪ task keys`
      // minus the suppressed ones. Deliberately not the suppressed names
      // themselves — those are hidden as *record* keys, and a form field that
      // happens to share a name with internal plumbing is still a real setting
      // that would otherwise disappear from this page altogether.
      const rendered = [
        ...columns.map((col) => col.key),
        ...Object.keys(task),
      ].filter((key) => !suppressedFields.has(key));
      return new Set<string>([
        ...rendered,
        // The record spells it `name` and the form spells it `task_name`, so
        // the key overlap above does not catch it. Every task-style app
        // inherits the field from the framework's own task form model, which is
        // what keeps naming it here app-agnostic.
        //
        // It is the only such mismatch today. A second one would want a
        // declared record-key/form-field alias map rather than another literal
        // here — the failure mode is a quietly duplicated field, which nothing
        // types or tests would catch.
        'task_name',
      ]);
    },
    // Derived from the same inputs `extraEntries` is, rather than from
    // `extraEntries` itself: that array is rebuilt every render, and depending
    // on it would defeat the memo the configuration card keys off.
    [suppressedFields, columns, task]
  );

  // Selected here rather than inside the card so the raw-document disclosure
  // below can key off whether this produced anything. Collapsing on "a stored
  // form exists" instead would leave a task that configured nothing beyond its
  // defaults showing a lone "Show raw configuration" button and no summary at
  // all — strictly worse than the always-visible sections this replaced.
  const configuredSettings = useMemo(
    () =>
      configurationSections && storedForm
        ? selectConfiguredSettings(
            configurationSections,
            storedForm,
            configurationExcludedNames
          )
        : [],
    [configurationSections, storedForm, configurationExcludedNames]
  );

  return (
    <>
      {connectivityWarning !== null &&
        connectivityWarning !== undefined &&
        typeof connectivityWarning === 'object' && (
          <ConnectivityWarningAlert
            warning={
              connectivityWarning as SepComponents['schemas']['framework__ConnectivityWarning']
            }
          />
        )}

      {/* The last run sits above everything else on the Overview: when a
          nightly backup fails, why it failed is the first thing the page owes
          the reader. Before this the Overview said nothing about any run, and
          the only failure signal was a banner riding router state that was
          gone after a reload. */}
      {taskName && (
        <LastRunCard
          taskNames={taskName}
          onOpenRun={() => setLastRunOpen(true)}
        />
      )}

      {/* Opened by name, not with the row the card is holding: the drawer then
          keeps following the run, so one watched from here reaches its terminal
          status in place. */}
      {lastRunOpen && taskName && (
        <TaskRunDetailDrawer
          open
          onClose={() => setLastRunOpen(false)}
          taskNames={taskName}
          taskLabel={taskName}
        />
      )}

      {/* Schedule / next-run sits first so it is visible without scrolling
          past the Task information card. Gate unchanged: plugins without the
          scheduling capability render nothing here, so their Overview is
          untouched. */}
      {schema.capabilities?.scheduling &&
        pluginName &&
        taskName &&
        scheduleHref && (
          <ScheduleSummary
            pluginName={pluginName}
            taskName={taskName}
            scheduleHref={scheduleHref}
          />
        )}

      <SectionCard title="Task information">
        <Grid container spacing={2}>
          {visibleColumns.map((col) => (
            <TaskOverviewDetailField
              key={col.key}
              label={col.label}
              value={task[col.key]}
              valueLabels={col.value_labels}
            />
          ))}
          {extraEntries.map(([key, value]) => (
            <TaskOverviewDetailField
              key={key}
              label={formatLabel(key)}
              value={value}
            />
          ))}
        </Grid>
      </SectionCard>

      {schema.capabilities?.stats && (
        <StatsCard
          taskName={
            typeof task.name === 'string' && task.name.trim()
              ? task.name.trim()
              : undefined
          }
        />
      )}

      {schema.capabilities?.pii_anonymization && (
        <SectionCard title="PII Anonymization">
          {Array.isArray(task.anonymized_entities) &&
          task.anonymized_entities.length > 0 ? (
            <Stack
              direction="row"
              spacing={1}
              sx={{ flexWrap: 'wrap', gap: 1 }}
            >
              {(task.anonymized_entities as string[]).map((entity) => (
                <Chip
                  key={entity}
                  label={entity.replace(/_/g, ' ')}
                  size="small"
                  data-testid="pii-entity-chip"
                />
              ))}
            </Stack>
          ) : (
            <Typography variant="body2" color="text.secondary">
              No PII entities configured for anonymization.
            </Typography>
          )}
        </SectionCard>
      )}

      {configuredSettings.length > 0 && (
        <TaskConfigurationCard configured={configuredSettings} />
      )}

      {plainDetailSections.map((section, idx) => (
        <DetailViewSectionCard
          key={`${section.title}:${idx}`}
          section={section}
          task={task}
        />
      ))}

      {rawDetailSections.length > 0 && (
        // A section of nothing but highlighted documents is the generated
        // config — authoritative, and the thing to read when the question is
        // what was actually emitted, but not what someone checking a backup
        // opens the page for. Demoted behind a disclosure once the
        // configuration card above can answer that question instead, and left
        // expanded for a task with no stored form, where it is the only account
        // of the configuration there is.
        <RawDetailDisclosure collapsed={configuredSettings.length > 0}>
          {rawDetailSections.map((section, idx) => (
            <DetailViewSectionCard
              key={`${section.title}:${idx}`}
              section={section}
              task={task}
            />
          ))}
        </RawDetailDisclosure>
      )}

      {children}
    </>
  );
}

interface LogsTabProps {
  taskNames: string[];
}

function LogsTab({ taskNames }: LogsTabProps) {
  const historyQuery = useTaskHistoryByNames(taskNames);
  const stop = useStopTaskHistory();
  const [openedRow, setOpenedRow] = useState<TaskHistoryEntry | null>(null);
  // Re-read the opened run out of the live query rather than showing the row as
  // it looked when it was clicked: this query already polls while anything is
  // running, so a backup watched from here flips to failed in place instead of
  // counting elapsed time forever behind a drawer that has to be reopened.
  // Falls back to the clicked row for a run with no id, and for one that has
  // since paged out of the list.
  const logsEntry = useMemo(
    () => resolveOpenedRun(openedRow, historyQuery.data?.items),
    [openedRow, historyQuery.data]
  );
  const logsTaskName = logsEntry?.task?.name ?? taskNames[0] ?? 'task';

  return (
    <>
      {historyQuery.error ? (
        <Alert severity="error" sx={{ mb: 3 }}>
          Failed to load execution history: {historyQuery.error.message}
        </Alert>
      ) : (
        <Paper variant="outlined" sx={{ p: 0, mb: 3 }}>
          <TaskHistoryTable
            data={historyQuery.data?.items ?? []}
            isLoading={historyQuery.isLoading}
            hideTaskNameColumn={taskNames.length <= 1}
            onViewLogs={setOpenedRow}
            onStopTask={(entry) => {
              if (entry.id !== null && entry.id !== undefined) {
                stop.mutate(entry.id);
              }
            }}
            isStopping={stop.isPending}
            actionError={stop.error}
            onDismissActionError={stop.reset}
          />
        </Paper>
      )}

      {logsEntry !== null && (
        <TaskRunDetailDrawer
          open
          onClose={() => setOpenedRow(null)}
          entry={logsEntry}
          taskLabel={logsTaskName}
        />
      )}
    </>
  );
}

interface ActionBarProps {
  schema: PluginSchema;
  pluginName: string;
  routeBase: string;
  taskName: string;
  executeActions?: TaskExecuteAction[];
  /** Whether the task carries a stored create-form body, enabling in-place edit. */
  hasStoredForm: boolean;
}

function emptyChain(): ChainValue {
  return { chain_task_names: [], chain_on_failure: false };
}

function ActionBar({
  schema,
  pluginName,
  routeBase,
  taskName,
  executeActions,
  hasStoredForm,
}: ActionBarProps) {
  const navigate = useNavigate();
  const { canMutate } = useAuth();
  const { enqueueSnackbar } = useSnackbar();
  const deleteTask = useDeletePluginTask(pluginName);
  const executeTask = useExecuteTask(pluginName);
  // Both actions are confirmed in a dialog that closes before the request
  // settles, so the failure is held here and rendered on the page behind it.
  const actionError = useActionError();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [pendingExecute, setPendingExecute] =
    useState<TaskExecuteAction | null>(null);
  const [chain, setChain] = useState<ChainValue>(emptyChain);
  /** Which execute action the current `chain` was composed for. */
  const [chainActionKey, setChainActionKey] = useState<string | null>(null);

  const chainingEnabled = !!schema.capabilities?.chaining;
  const {
    data: pluginTasksData,
    isLoading: pluginTasksLoading,
    isError: pluginTasksError,
    error: pluginTasksLoadError,
  } = usePluginTasks<{ name: string }>(pluginName, undefined, {
    fetchAllPages: true,
    enabled: chainingEnabled,
  });
  const availableTasks = useMemo(
    () => (pluginTasksData?.items ?? []).map((t) => ({ name: t.name })),
    [pluginTasksData]
  );

  useEffect(() => {
    // Opening a different action starts from an empty chain; reopening the same
    // one keeps what was composed, so a refused execute can be retried without
    // rebuilding the chain. Closing the dialog (which now happens on failure
    // too) leaves the chain alone for that retry.
    if (!pendingExecute) {
      return;
    }
    const key = `${pendingExecute.taskName}\u0000${pendingExecute.label}`;
    if (key !== chainActionKey) {
      setChain(emptyChain());
      setChainActionKey(key);
    }
  }, [pendingExecute, chainActionKey]);

  const resolvedExecuteActions =
    executeActions ??
    ([
      {
        label: 'Execute',
        taskName,
        testId: 'plugin-task-execute',
      },
    ] satisfies TaskExecuteAction[]);

  const handleExecute = async () => {
    if (!pendingExecute) {
      return;
    }
    const hasChain = chain.chain_task_names.length > 0;
    let executeBody = pendingExecute.executeBody;
    if (hasChain) {
      executeBody = {
        ...pendingExecute.executeBody,
        chain_task_names: chain.chain_task_names,
        chain_on_failure: chain.chain_on_failure,
      };
    }
    actionError.clearError();
    try {
      const executeArgs = executeBody
        ? { taskName: pendingExecute.taskName, executeBody }
        : { taskName: pendingExecute.taskName };
      await executeTask.mutateAsync(executeArgs);
      enqueueSnackbar(
        `${schema.display_name} task "${pendingExecute.taskName}" started`,
        {
          variant: 'success',
        }
      );
    } catch (e) {
      actionError.reportError(e);
    } finally {
      // Close on confirm whatever the outcome, like the adjacent delete: a
      // dialog left open holding a failure hides the message rendered behind it.
      setPendingExecute(null);
    }
  };

  const handleDelete = async () => {
    actionError.clearError();
    try {
      await deleteTask.mutateAsync(taskName);
      enqueueSnackbar(`${schema.display_name} task deleted`, {
        variant: 'success',
      });
      // Anchor to the plugin root explicitly. Relative `..` chains depend
      // on which tab the user is on (Overview vs. Execution History renders a deeper
      // sub-route via nested `<Routes>`), so use an absolute path.
      navigate(routeBase);
    } catch (e) {
      actionError.reportError(e);
    } finally {
      setConfirmOpen(false);
    }
  };

  const editUnavailable =
    "Editing isn't available for this task — it has no saved form input.";

  // Every action here is a mutation except Schedule (navigation), so a
  // read-only session with no scheduling capability is left with no bar at all.
  if (!canMutate && !schema.capabilities?.scheduling) {
    return null;
  }

  return (
    <>
      <Stack direction="row" spacing={1} sx={{ mb: 3 }}>
        {schema.capabilities?.scheduling && (
          <Button
            variant="outlined"
            startIcon={<ScheduleIcon />}
            onClick={() => navigate(`${routeBase}/schedule`)}
            data-testid="plugin-task-schedule"
          >
            Schedule
          </Button>
        )}

        {canMutate &&
          resolvedExecuteActions.map((action) => (
            <Button
              key={`${action.taskName}-${action.label}`}
              variant="outlined"
              startIcon={<PlayArrowIcon />}
              onClick={() => setPendingExecute(action)}
              disabled={executeTask.isPending}
              data-testid={action.testId ?? 'plugin-task-execute'}
            >
              {action.label}
            </Button>
          ))}

        {canMutate &&
          (hasStoredForm ? (
            <Button
              variant="outlined"
              component={Link}
              to={`${routeBase}/task/${encodeURIComponent(taskName)}/edit`}
              startIcon={<EditIcon />}
              data-testid="plugin-task-edit"
            >
              Edit
            </Button>
          ) : (
            <Tooltip title={editUnavailable}>
              <span>
                <Button
                  variant="outlined"
                  startIcon={<EditIcon />}
                  disabled
                  data-testid="plugin-task-edit"
                >
                  Edit
                </Button>
              </span>
            </Tooltip>
          ))}

        <Box sx={{ flexGrow: 1 }} />

        {canMutate && (
          <Button
            variant="outlined"
            startIcon={<DeleteIcon />}
            onClick={() => setConfirmOpen(true)}
            disabled={deleteTask.isPending}
            data-testid="plugin-task-delete"
          >
            Delete
          </Button>
        )}
      </Stack>

      <ActionErrorAlert
        error={actionError.error}
        onClose={actionError.clearError}
        sx={{ mb: 3 }}
        testId="plugin-task-action-error"
      />

      <Dialog
        open={pendingExecute !== null}
        onClose={() => {
          if (!executeTask.isPending) {
            setPendingExecute(null);
          }
        }}
        fullWidth
        maxWidth="sm"
      >
        <DialogTitle>
          {pendingExecute?.label ?? 'Execute'} {schema.display_name} task?
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {pendingExecute?.confirmMessage ??
              `Are you sure you want to execute the task ${pendingExecute?.taskName ?? taskName} now?`}
          </DialogContentText>
          {chainingEnabled && pendingExecute && (
            <Box sx={{ mt: 2 }}>
              {pluginTasksLoading ? (
                <Box sx={{ display: 'flex', justifyContent: 'center' }}>
                  <CircularProgress
                    size={24}
                    data-testid="chain-tasks-loading"
                  />
                </Box>
              ) : pluginTasksError ? (
                <Alert severity="error" data-testid="chain-tasks-error">
                  Couldn&apos;t load tasks available to chain
                  {pluginTasksLoadError instanceof Error
                    ? `: ${pluginTasksLoadError.message}`
                    : ''}
                </Alert>
              ) : (
                <ChainBuilder
                  availableTasks={availableTasks}
                  currentTaskName={pendingExecute.taskName}
                  value={chain}
                  onChange={setChain}
                  disabled={executeTask.isPending}
                />
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setPendingExecute(null)}
            disabled={executeTask.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleExecute}
            variant="contained"
            disabled={executeTask.isPending}
            data-testid="plugin-task-execute-confirm"
          >
            {pendingExecute?.label ?? 'Execute'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={confirmOpen} onClose={() => setConfirmOpen(false)}>
        <DialogTitle>Delete {schema.display_name} task?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            This will permanently remove the task definition. Past run history
            is unaffected.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setConfirmOpen(false)}
            disabled={deleteTask.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleDelete}
            variant="contained"
            disabled={deleteTask.isPending}
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

export function PluginDetailPage({
  schema,
  pluginName,
  routeBase: routeBaseProp,
  mockTasks,
  mockEntityItems,
  browseOnly = false,
  suppressDetailKeys = [],
  getTaskExecuteActions,
  getTaskHistoryNames,
  renderTaskDetailChildren,
  renderEntityDetailChildren,
  detailEntityName,
  detailIdParam,
  resolveParentPath,
  hideDetailChrome = false,
  allowListEntityDelete = false,
}: PluginDetailPageProps) {
  const routeBase = resolvePluginRouteBase(pluginName, routeBaseProp);
  const { canMutate } = useAuth();
  const params = useParams<Record<string, string | undefined>>();
  const id = (detailIdParam && params[detailIdParam]) ?? params.id;
  const entityName = detailEntityName ?? params.entityName;
  const splat = params['*'];
  const navigate = useNavigate();
  const location = useLocation();
  const entitySchema = useMemo(
    () =>
      schema.entities?.find((e: PluginEntitySchema) => e.name === entityName),
    [schema.entities, entityName]
  );
  const multi = Boolean(schema.entities?.length && entityName && entitySchema);

  const customParentPath = useMemo(() => {
    if (!resolveParentPath) {
      return null;
    }
    return resolveParentPath(location.pathname);
  }, [location.pathname, resolveParentPath]);

  const taskQuery = usePluginTask(pluginName, id, mockTasks, {
    enabled: !multi && Boolean(id),
  });
  const entityQuery = usePluginEntityDetail(
    pluginName,
    entityName ?? '',
    id,
    multi ? mockEntityItems?.[entityName!] : undefined,
    { enabled: multi && Boolean(id) }
  );

  const { data: task, isLoading } = multi ? entityQuery : taskQuery;
  const listView = multi ? entitySchema!.list_view : schema.list_view!;
  const title = multi ? entitySchema!.display_name : schema.display_name;
  const headingWhenChromeHidden = useMemo(
    () =>
      hideDetailChrome && multi
        ? detailScreenHeading(entityName, entitySchema?.display_name)
        : null,
    [hideDetailChrome, multi, entityName, entitySchema?.display_name]
  );

  const deleteEntity = useDeletePluginEntity(
    pluginName,
    entityName ?? '',
    multi ? mockEntityItems?.[entityName!] : undefined
  );

  const [entityDeleteOpen, setEntityDeleteOpen] = useState(false);
  // The confirm dialog closes on confirm, so a refusal has to land on the
  // detail page the user is returned to.
  const deleteEntityError = useActionError();

  const confirmEntityDelete = () => {
    setEntityDeleteOpen(false);
    if (!id || !multi) {
      return;
    }
    deleteEntityError.clearError();
    deleteEntity.mutate(id, {
      onError: (error) => deleteEntityError.reportError(error),
      onSuccess: () =>
        customParentPath
          ? navigate(customParentPath)
          : entityName
            ? navigate(pathToEntityList(location.pathname, entityName))
            : navigate('..', { relative: 'path' }),
    });
  };

  if (multi) {
    if (isLoading) {
      return (
        <Box>
          {headingWhenChromeHidden ? (
            <Typography
              component="h1"
              variant="h4"
              sx={{ mb: 2, fontWeight: 600 }}
            >
              {headingWhenChromeHidden}
            </Typography>
          ) : (
            <Skeleton variant="text" width={300} height={40} />
          )}
          <Skeleton variant="rectangular" height={200} sx={{ mt: 2 }} />
        </Box>
      );
    }

    if (!task) {
      return (
        <Box>
          {headingWhenChromeHidden ? (
            <Typography
              component="h1"
              variant="h4"
              sx={{ mb: 2, fontWeight: 600 }}
            >
              {headingWhenChromeHidden}
            </Typography>
          ) : null}
          <Typography variant="h5">Not found</Typography>
        </Box>
      );
    }

    const recordName =
      typeof task.name === 'string' && task.name.trim()
        ? task.name.trim()
        : typeof task.name === 'number'
          ? String(task.name)
          : undefined;

    return (
      <Box>
        <DeleteConfirmDialog
          open={entityDeleteOpen}
          onClose={() => setEntityDeleteOpen(false)}
          onConfirm={confirmEntityDelete}
          title={`Delete from ${schema.display_name}?`}
          description={
            recordName
              ? `Permanently delete ${title} "${recordName}" (id ${id}) from ${schema.display_name}? This cannot be undone.`
              : `Permanently delete ${title} (id ${id}) from ${schema.display_name}? This cannot be undone.`
          }
        />

        <ActionErrorAlert
          error={deleteEntityError.error}
          onClose={deleteEntityError.clearError}
          sx={{ mb: 2 }}
          testId="entity-detail-action-error"
        />

        {headingWhenChromeHidden ? (
          <Typography
            component="h1"
            variant="h4"
            sx={{ mb: 2, fontWeight: 600 }}
          >
            {headingWhenChromeHidden}
          </Typography>
        ) : null}
        {!hideDetailChrome && (
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1,
              mb: 3,
              flexWrap: 'wrap',
            }}
          >
            <IconButton
              onClick={() =>
                customParentPath
                  ? navigate(customParentPath)
                  : multi && entityName
                    ? navigate(pathToEntityList(location.pathname, entityName))
                    : navigate('..', { relative: 'path' })
              }
            >
              <ArrowBackIcon />
            </IconButton>
            <Typography variant="h4">
              {title} #{id}
            </Typography>
            {isTaskHistoryStatus(task.status) ? (
              <TaskHistoryStatusBadge status={task.status} />
            ) : typeof task.status === 'string' ? (
              // Unrecognized string status (unexpected/older value): fall back to
              // a plain chip so status never silently disappears, matching
              // SchemaListView's status-cell fallback.
              <Chip label={task.status} size="small" />
            ) : null}
            {multi && !browseOnly && canMutate && (
              <>
                <Button
                  component={Link}
                  to="edit"
                  relative="path"
                  startIcon={<EditIcon />}
                  variant="outlined"
                  size="small"
                >
                  Edit
                </Button>
                <Button
                  variant="outlined"
                  size="small"
                  onClick={() => setEntityDeleteOpen(true)}
                  disabled={deleteEntity.isPending}
                >
                  Delete
                </Button>
              </>
            )}
          </Box>
        )}

        <Paper sx={{ p: 2 }}>
          <Grid container spacing={2}>
            {listView.columns
              .filter(
                (col) =>
                  col.format !== 'actions' &&
                  col.key !== '_actions' &&
                  !(listView.overview_hidden_fields ?? []).includes(col.key)
              )
              .map((col) => (
                <EntityDetailField
                  key={col.key}
                  highlightLanguage={entitySchema?.detail_highlights?.[col.key]}
                  label={col.label}
                  value={task[col.key]}
                  valueLabels={col.value_labels}
                />
              ))}

            {Object.entries(task)
              .filter(
                ([key]) =>
                  !listView.columns.some((c) => c.key === key) &&
                  key !== 'id' &&
                  key !== '_actions' &&
                  !suppressDetailKeys.includes(key) &&
                  !(listView.overview_hidden_fields ?? []).includes(key)
              )
              .map(([key, value]) => (
                <EntityDetailField
                  key={key}
                  highlightLanguage={entitySchema?.detail_highlights?.[key]}
                  label={key}
                  value={value}
                />
              ))}
          </Grid>
        </Paper>

        {multi &&
          entityName &&
          renderEntityDetailChildren &&
          renderEntityDetailChildren({
            entityName,
            record: task as Record<string, unknown>,
            schema,
            pathname: location.pathname,
            pluginName,
            mockEntityItems,
            allowListEntityDelete,
          })}
      </Box>
    );
  }

  const tabValue = resolveTabFromSplat(splat);

  if (isLoading) {
    return (
      <Box>
        <Skeleton variant="text" width={300} height={40} />
        <Skeleton variant="rectangular" height={200} sx={{ mt: 2 }} />
      </Box>
    );
  }

  if (!task || !id) {
    return (
      <Box>
        <Typography variant="h5">Task not found</Typography>
      </Box>
    );
  }

  const taskName = typeof task.name === 'string' ? task.name : id;
  const hasStoredForm = Boolean(getStoredForm(task as Record<string, unknown>));
  const detailBase = `${routeBase}/task/${encodeURIComponent(id)}`;
  const taskExecuteActions = getTaskExecuteActions?.(
    task as Record<string, unknown>
  );
  const taskHistoryNames =
    getTaskHistoryNames?.(task as Record<string, unknown>) ??
    (taskName ? [taskName] : []);

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
        <IconButton
          onClick={() => navigate(routeBase)}
          aria-label="Back to list"
        >
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="overline" color="text.secondary">
          {schema.display_name}
        </Typography>
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 3, ml: 5 }}>
        <Typography variant="h4">{taskName}</Typography>
        {isTaskHistoryStatus(task.status) ? (
          <TaskHistoryStatusBadge status={task.status} />
        ) : typeof task.status === 'string' ? (
          // Unrecognized string status (unexpected/older value): fall back to a
          // plain chip so status never silently disappears, matching
          // SchemaListView's status-cell fallback.
          <Chip label={task.status} size="small" />
        ) : null}
      </Box>

      <ActionBar
        schema={schema}
        pluginName={pluginName}
        routeBase={routeBase}
        taskName={taskName}
        executeActions={taskExecuteActions}
        hasStoredForm={hasStoredForm}
      />

      <Tabs
        value={tabValue}
        sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}
      >
        <Tab
          label="Overview"
          value="overview"
          component={Link}
          to={detailBase}
          replace
        />
        <Tab
          label="Execution History"
          value="logs"
          component={Link}
          to={`${detailBase}/logs`}
          replace
        />
      </Tabs>

      <Routes>
        <Route
          index
          element={
            <OverviewTab
              schema={schema}
              task={task}
              hiddenFields={suppressDetailKeys}
              pluginName={pluginName}
              scheduleHref={`${routeBase}/schedule`}
            >
              {renderTaskDetailChildren?.({
                task: task as Record<string, unknown>,
                pluginName,
                schema,
              })}
            </OverviewTab>
          }
        />
        <Route path="logs" element={<LogsTab taskNames={taskHistoryNames} />} />
      </Routes>
    </Box>
  );
}
