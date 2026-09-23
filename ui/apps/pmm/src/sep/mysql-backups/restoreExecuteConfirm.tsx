import { ReactNode } from 'react';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { PluginField, PluginSchema } from '@sep/api';
import {
  getStoredForm,
  useSchemas,
  type TaskExecuteAction,
} from '@sep/framework';

const UNKNOWN = 'Not set';
const SAME_AS_BACKUP = 'Same databases as in the backup';

type BackupType = 'M' | 'X' | 'B';

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

function asOptionalBoolean(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined;
}

function asBackupType(value: unknown): BackupType | undefined {
  return value === 'M' || value === 'X' || value === 'B' ? value : undefined;
}

/** The destination service name the side-car stamps into the task's meta. */
function getServiceName(task: Record<string, unknown>): string | undefined {
  const data = task.data;
  if (typeof data !== 'object' || data === null) {
    return undefined;
  }
  const meta = (data as { meta?: unknown }).meta;
  if (typeof meta !== 'object' || meta === null) {
    return undefined;
  }
  return asDisplayString((meta as Record<string, unknown>)._service_name);
}

function asInventoryId(value: unknown): number | undefined {
  const id =
    typeof value === 'number'
      ? value
      : typeof value === 'string' && /^\d+$/.test(value)
        ? Number(value)
        : undefined;
  return id !== undefined && Number.isInteger(id) && id > 0 ? id : undefined;
}

/** True when this plugin name is a MySQL restore related app. */
export function isMysqlRestorePluginName(pluginName: string): boolean {
  return pluginName.includes('restore');
}

/** True when this plugin (or task payload) is a MySQL restore. */
export function isMysqlRestoreTask(
  task: Record<string, unknown>,
  pluginName?: string
): boolean {
  if (pluginName && isMysqlRestorePluginName(pluginName)) {
    return true;
  }
  const form = getStoredForm(task);
  return Boolean(
    form && ('overwrite_tables' in form || 'backup_source' in form)
  );
}

export interface MysqlRestoreConfirmDetails {
  source: string;
  backupType?: BackupType;
  serviceName?: string;
  executorHost: string;
  targetHost: string;
  targetDatabase: string;
  targetSchema?: { serviceId: number; schemaId: number };
  overwriteTables: boolean | undefined;
  restoreMycnf?: boolean;
}

function resolveTargetDatabase(
  form: Record<string, unknown> | undefined
): Pick<MysqlRestoreConfirmDetails, 'targetDatabase' | 'targetSchema'> {
  if (!form) {
    return { targetDatabase: UNKNOWN };
  }
  const schemaId = asInventoryId(form.schema_id);
  if (schemaId === undefined) {
    const hydratedName =
      typeof form.schema_id === 'object'
        ? asDisplayString(form.schema_id)
        : undefined;
    return { targetDatabase: hydratedName ?? SAME_AS_BACKUP };
  }
  const targetDatabase = `Unknown (inventory ID ${schemaId})`;
  const serviceId = asInventoryId(form.service_id);
  return serviceId === undefined
    ? { targetDatabase }
    : { targetDatabase, targetSchema: { serviceId, schemaId } };
}

/** Pull source / target / overwrite facts from the task detail + stored form. */
export function getMysqlRestoreConfirmDetails(
  task: Record<string, unknown>
): MysqlRestoreConfirmDetails {
  const form = getStoredForm(task);

  const source = asDisplayString(form?.backup_source) ?? UNKNOWN;
  const backupType = asBackupType(task.backup_type ?? form?.backup_type);

  // `host` / `port` are the MyDumper destination (RestoresResponse / DEST_*).
  // `hostname` is the executor (pmm-agent), where XtraBackup and Binlog restore.
  const executorHost =
    asDisplayString(task.hostname) ??
    asDisplayString(form?.hostname) ??
    UNKNOWN;
  const host =
    asDisplayString(task.host) ??
    asDisplayString(form?.host) ??
    asDisplayString(form?.dest_host);
  const port =
    asDisplayString(task.port) ??
    asDisplayString(form?.dest_port) ??
    asDisplayString(form?.port);
  const targetHost = host && port ? `${host}:${port}` : (host ?? UNKNOWN);

  const overwriteTables = asOptionalBoolean(form?.overwrite_tables);

  return {
    source,
    backupType,
    serviceName: getServiceName(task),
    executorHost,
    targetHost,
    ...resolveTargetDatabase(form),
    overwriteTables,
    restoreMycnf: asOptionalBoolean(form?.restore_mycnf),
  };
}

function schemaField(
  schema: PluginSchema | undefined,
  name: string
): PluginField | undefined {
  for (const section of schema?.forms ?? []) {
    for (const field of section.fields) {
      if (field.type !== 'one_of' && field.name === name) {
        return field;
      }
    }
  }
  return undefined;
}

/**
 * The consequence text the restore schema attaches to a field it marks
 * destructive. Quoted rather than paraphrased, so the dialog cannot drift from
 * what the side-car says that field destroys.
 */
function schemaConsequence(
  schema: PluginSchema | undefined,
  name: string
): string | undefined {
  return schemaField(schema, name)?.destructive || undefined;
}

function restoresOnExecutor(backupType: BackupType | undefined): boolean {
  return backupType === 'X' || backupType === 'B';
}

function liveTarget(details: MysqlRestoreConfirmDetails): string {
  if (restoresOnExecutor(details.backupType)) {
    return details.executorHost === UNKNOWN
      ? 'the live MySQL server on the execution host'
      : `the live MySQL server on ${details.executorHost}`;
  }
  const address =
    details.targetHost === UNKNOWN ? undefined : details.targetHost;
  if (details.serviceName) {
    return address
      ? `the live database service ${details.serviceName} (${address})`
      : `the live database service ${details.serviceName}`;
  }
  return address
    ? `the live database service at ${address}`
    : 'the live destination database service';
}

function LiveDataAlert({ details }: { details: MysqlRestoreConfirmDetails }) {
  // With Overwrite tables off, myloader refuses a table that already exists
  // instead of dropping it, so claiming existing data is lost would contradict
  // the schema's own description of that option, shown beneath it.
  const keepsExistingTables =
    !restoresOnExecutor(details.backupType) &&
    details.overwriteTables === false;
  return (
    <Alert severity="error" data-testid="mysql-restore-live-data-alert">
      <AlertTitle>This restore writes into live data</AlertTitle>
      It restores the backup into {liveTarget(details)}, and a restore cannot be
      undone.
      {keepsExistingTables
        ? null
        : ' Data already there can be overwritten or lost.'}
    </Alert>
  );
}

function ConfirmRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <Typography component="div" variant="body2">
      <Typography component="span" variant="body2" color="text.secondary">
        {label}:{' '}
      </Typography>
      {value}
    </Typography>
  );
}

function InventoryDatabaseName({
  serviceId,
  schemaId,
  fallback,
}: {
  serviceId: number;
  schemaId: number;
  fallback: string;
}) {
  const { data, isLoading } = useSchemas({ serviceId });
  if (isLoading) {
    return <>Loading…</>;
  }
  return (
    <>{data?.find((schema) => schema.id === schemaId)?.name ?? fallback}</>
  );
}

function overwriteLabel(overwriteTables: boolean | undefined): string {
  if (overwriteTables === undefined) {
    return 'Unknown';
  }
  return overwriteTables ? 'Yes' : 'No';
}

function MydumperTargetRows({
  details,
  schema,
}: {
  details: MysqlRestoreConfirmDetails;
  schema?: PluginSchema;
}) {
  return (
    <>
      <ConfirmRow label="Target host" value={details.targetHost} />
      <ConfirmRow
        label="Target database"
        value={
          details.targetSchema ? (
            <InventoryDatabaseName
              serviceId={details.targetSchema.serviceId}
              schemaId={details.targetSchema.schemaId}
              fallback={details.targetDatabase}
            />
          ) : (
            details.targetDatabase
          )
        }
      />
      <ConfirmRow
        label="Overwrite tables"
        value={overwriteLabel(details.overwriteTables)}
      />
      {details.overwriteTables === false ? (
        <Typography
          variant="caption"
          color="text.secondary"
          data-testid="mysql-restore-overwrite-explanation"
        >
          {schemaField(schema, 'overwrite_tables')?.description ??
            'This only decides whether tables that already exist are replaced. The restore writes into the live database either way.'}
        </Typography>
      ) : null}
    </>
  );
}

function MydumperWarnings({
  overwriteTables,
  schema,
}: {
  overwriteTables: boolean | undefined;
  schema?: PluginSchema;
}) {
  if (overwriteTables === true) {
    return (
      <Alert severity="warning" data-testid="mysql-restore-overwrite-alert">
        {schemaConsequence(schema, 'overwrite_tables') ??
          'Existing tables on the target database will be overwritten.'}
      </Alert>
    );
  }
  if (overwriteTables === undefined) {
    return (
      <Alert
        severity="warning"
        data-testid="mysql-restore-overwrite-unknown-alert"
      >
        Could not determine whether existing tables on the target database will
        be overwritten. Treat this restore as destructive.
      </Alert>
    );
  }
  return null;
}

/**
 * Rich confirm body: a destructive warning naming the live target, then the
 * source, target and overwrite facts, then the schema's own consequence text
 * for each destructive option this restore turns on.
 */
export function MysqlRestoreConfirmContent({
  details,
  schema,
}: {
  details: MysqlRestoreConfirmDetails;
  schema?: PluginSchema;
}): ReactNode {
  const onExecutor = restoresOnExecutor(details.backupType);

  return (
    <Stack spacing={1.5} data-testid="mysql-restore-execute-confirm">
      <LiveDataAlert details={details} />
      <Stack spacing={0.5}>
        <ConfirmRow label="Source backup" value={details.source} />
        {onExecutor ? (
          <ConfirmRow label="Target host" value={details.executorHost} />
        ) : (
          <MydumperTargetRows details={details} schema={schema} />
        )}
      </Stack>
      {details.backupType === 'X' ? (
        <Alert severity="warning" data-testid="mysql-restore-datadir-alert">
          MySQL on the target host is stopped during the restore.{' '}
          {schemaConsequence(schema, 'datadir') ??
            'Its data directory is replaced with the backup, so every database on that server is overwritten.'}
        </Alert>
      ) : null}
      {details.backupType === 'X' && details.restoreMycnf === true ? (
        <Alert severity="warning" data-testid="mysql-restore-mycnf-alert">
          {schemaConsequence(schema, 'restore_mycnf') ??
            'The configuration files saved in the backup are written over the live ones.'}
        </Alert>
      ) : null}
      {details.backupType === 'B' ? (
        <Alert severity="warning" data-testid="mysql-restore-binlog-alert">
          The backup&apos;s binary logs are replayed into the MySQL server on
          the target host.
        </Alert>
      ) : null}
      {onExecutor ? null : (
        <MydumperWarnings
          overwriteTables={details.overwriteTables}
          schema={schema}
        />
      )}
    </Stack>
  );
}

/**
 * Custom execute actions for MySQL Backups / Restores.
 * Restores get a destructive confirmation that names the live target and
 * quotes the schema's consequence text; backup tasks fall through to the
 * framework default Execute button.
 */
export function getMysqlBackupsTaskExecuteActions(
  task: Record<string, unknown>,
  context?: { pluginName: string; schema?: PluginSchema }
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
      destructive: true,
      confirmContent: (
        <MysqlRestoreConfirmContent
          details={details}
          schema={context?.schema}
        />
      ),
    },
  ];
}
