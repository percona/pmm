import { ReactNode } from 'react';
import Alert from '@mui/material/Alert';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { getStoredForm, type TaskExecuteAction } from '@sep/framework';

const UNKNOWN = 'Not set';

function asDisplayString(value: unknown): string | undefined {
  if (value === null || value === undefined || value === '') {
    return undefined;
  }
  if (typeof value === 'string' || typeof value === 'number') {
    return String(value);
  }
  // Hydrated ServiceRef / SchemaRef option from a live form
  if (
    typeof value === 'object' &&
    value !== null &&
    'name' in value &&
    typeof (value as { name: unknown }).name === 'string' &&
    (value as { name: string }).name
  ) {
    return (value as { name: string }).name;
  }
  return undefined;
}

function asBoolean(value: unknown): boolean {
  return value === true;
}

/** True when this plugin (or task payload) is a MySQL restore. */
export function isMysqlRestoreTask(
  task: Record<string, unknown>,
  pluginName?: string
): boolean {
  if (pluginName?.includes('restore')) {
    return true;
  }
  const form = getStoredForm(task);
  if (form && ('overwrite_tables' in form || 'backup_source' in form)) {
    return true;
  }
  // PMM-15480 / newer RestoresResponse may surface overwrite on the task itself
  return 'overwrite_tables' in task && 'backup_source' in task;
}

export interface MysqlRestoreConfirmDetails {
  source: string;
  targetHost: string;
  targetDatabase: string;
  overwriteTables: boolean;
}

/** Pull source / target / overwrite facts from the task detail + stored form. */
export function getMysqlRestoreConfirmDetails(
  task: Record<string, unknown>
): MysqlRestoreConfirmDetails {
  const form = getStoredForm(task) ?? {};

  const source =
    asDisplayString(form.backup_source) ??
    asDisplayString(task.backup_source) ??
    UNKNOWN;

  // `host` / `port` are the restore destination (RestoresResponse / DEST_*).
  // `hostname` is the executor (pmm-agent) — never show it as "Target host".
  const host =
    asDisplayString(task.host) ??
    asDisplayString(form.host) ??
    asDisplayString(form.dest_host);
  const port =
    asDisplayString(task.port) ??
    asDisplayString(form.dest_port) ??
    asDisplayString(form.port);
  const targetHost =
    host && port ? `${host}:${port}` : (host ?? port ?? UNKNOWN);

  const targetDatabase =
    asDisplayString(form.schema_id) ??
    asDisplayString(form.database) ??
    asDisplayString(task.database) ??
    asDisplayString(task.schema_id) ??
    UNKNOWN;

  const overwriteTables = asBoolean(
    form.overwrite_tables ?? task.overwrite_tables
  );

  return { source, targetHost, targetDatabase, overwriteTables };
}

function ConfirmRow({ label, value }: { label: string; value: string }) {
  return (
    <Typography component="div" variant="body2">
      <Typography component="span" variant="body2" color="text.secondary">
        {label}:{' '}
      </Typography>
      {value}
    </Typography>
  );
}

/** Rich confirm body naming source, target, and overwrite behaviour. */
export function MysqlRestoreConfirmContent({
  details,
  mode = 'execute',
}: {
  details: MysqlRestoreConfirmDetails;
  mode?: 'execute' | 'schedule';
}): ReactNode {
  const intro =
    mode === 'schedule'
      ? 'You are about to schedule this restore. Confirm the target before continuing.'
      : 'You are about to run this restore. Confirm the target before continuing.';
  const testId =
    mode === 'schedule'
      ? 'mysql-restore-schedule-confirm'
      : 'mysql-restore-execute-confirm';

  return (
    <Stack spacing={1.5} data-testid={testId}>
      <Typography variant="body2">{intro}</Typography>
      <Stack spacing={0.5}>
        <ConfirmRow label="Source backup" value={details.source} />
        <ConfirmRow label="Target host" value={details.targetHost} />
        <ConfirmRow label="Target database" value={details.targetDatabase} />
        <ConfirmRow
          label="Overwrite tables"
          value={details.overwriteTables ? 'Yes' : 'No'}
        />
      </Stack>
      {details.overwriteTables ? (
        <Alert severity="warning" data-testid="mysql-restore-overwrite-alert">
          Existing tables on the target database will be overwritten.
        </Alert>
      ) : null}
    </Stack>
  );
}

/**
 * Custom execute actions for MySQL Backups / Restores.
 * Restores get a confirmation that names source, target, and overwrite;
 * backup tasks fall through to the framework default Execute button.
 */
export function getMysqlBackupsTaskExecuteActions(
  task: Record<string, unknown>,
  context?: { pluginName: string }
): TaskExecuteAction[] | undefined {
  if (!isMysqlRestoreTask(task, context?.pluginName)) {
    return undefined;
  }

  const details = getMysqlRestoreConfirmDetails(task);
  const taskName = typeof task.name === 'string' ? task.name : '';

  return [
    {
      label: 'Execute',
      taskName,
      testId: 'mysql-restore-execute',
      confirmContent: (
        <MysqlRestoreConfirmContent details={details} mode="execute" />
      ),
    },
  ];
}

/**
 * Schedule confirmation for MySQL Restores (product decision: restores stay
 * schedulable, with the same source/target/overwrite confirm as Execute).
 */
export function getMysqlBackupsScheduleWarning(
  taskName: string,
  context: {
    pluginName: string;
    tasks: Record<string, unknown>[];
  }
): ReactNode | undefined {
  if (!context.pluginName.includes('restore')) {
    return undefined;
  }

  const task = context.tasks.find((t) => t.name === taskName);
  if (!task) {
    // Incomplete poll / stale list: still block on confirm, but do not invent
    // a safe-looking "Overwrite: No" / "Not set" summary.
    return (
      <Stack
        spacing={1.5}
        data-testid="mysql-restore-schedule-confirm-incomplete"
      >
        <Alert severity="warning">
          Could not load details for restore task &apos;{taskName}&apos;. Open
          the task and verify the target host, database, and overwrite setting
          before scheduling.
        </Alert>
      </Stack>
    );
  }

  const details = getMysqlRestoreConfirmDetails(task);
  return <MysqlRestoreConfirmContent details={details} mode="schedule" />;
}
