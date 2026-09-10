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

  const host =
    asDisplayString(task.host) ??
    asDisplayString(form.host) ??
    asDisplayString(task.hostname) ??
    asDisplayString(form.hostname);
  const port =
    asDisplayString(task.port) ??
    asDisplayString(form.port) ??
    asDisplayString(form.dest_port);
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

/** Rich execute-dialog body naming source, target, and overwrite behaviour. */
export function MysqlRestoreExecuteConfirmContent({
  details,
}: {
  details: MysqlRestoreConfirmDetails;
}): ReactNode {
  return (
    <Stack spacing={1.5} data-testid="mysql-restore-execute-confirm">
      <Typography variant="body2">
        You are about to run this restore. Confirm the target before continuing.
      </Typography>
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
      confirmContent: <MysqlRestoreExecuteConfirmContent details={details} />,
    },
  ];
}
