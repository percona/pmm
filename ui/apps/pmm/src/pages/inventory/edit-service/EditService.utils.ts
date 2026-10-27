import { toServiceAgent } from 'api/inventory.utils';
import {
  AgentType,
  type RdsCredentials,
  type RdsExporter,
  type ServiceAgent,
} from 'types/agent.types';
import {
  type EditableService,
  type InventoryService,
  type ManagedServiceType,
  MetricsMode,
  type ServiceChanges,
  type UpdateServiceParams,
} from 'types/services.types';
import {
  DEFAULT_SLOWLOG_FILE_SIZE,
  EXPORTER_TYPES,
  MIB,
  NUMBER_SETTINGS,
  QUERY_SOURCES,
  SERVICE_SETTINGS,
} from './EditService.constants';
import type { EditServiceFormValues } from './EditService.schema';
import {
  QuerySource,
  RdsAuthMode,
  type SettingName,
} from './EditService.types';

export const toCustomLabelsText = (labels: Record<string, string>) =>
  Object.entries(labels)
    .map(([name, value]) => `${name}:${value}`)
    .join('\n');

// Only label names are restricted, so a value can hold colons too
export const splitCustomLabel = (entry: string): [string, string] => {
  const separator = entry.indexOf(':');

  return separator === -1
    ? [entry, '']
    : [entry.slice(0, separator), entry.slice(separator + 1)];
};

// Labels are separated by new lines or spaces, but a space only starts a new
// label before another "name:", so values can hold spaces
export const splitCustomLabelEntries = (text: string) =>
  text
    .split(/\n|\s+(?=[^\s:]+:)/)
    .map((entry) => entry.trim())
    .filter(Boolean);

export const parseCustomLabels = (text: string): Record<string, string> =>
  Object.fromEntries(splitCustomLabelEntries(text).map(splitCustomLabel));

export const toListText = (list: string[]) => list.join(', ');

export const parseList = (text: string) =>
  text
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);

// Seconds from a protobuf duration such as "1.500s", blank for the default
export const toSecondsText = (duration: string) => {
  const seconds = parseFloat(duration);

  return seconds > 0 ? String(seconds) : '';
};

// The server stores a disabled rotation as zero, which the API takes as a
// negative size
export const toSlowlogFileSizeText = (bytes: number) =>
  bytes === 0 ? '-1' : String(bytes / MIB);

export const toSlowlogFileSize = (text: string) => {
  const mib = Number(text);

  return mib < 0 ? -1 : mib * MIB;
};

// As pmm-admin and adding the exporter do
const toMetricsPath = (path: string) =>
  path && !path.startsWith('/') ? `/${path}` : path;

export const getRdsAuthMode = (exporter: RdsExporter | null) => {
  if (exporter?.awsRoleArn) {
    return RdsAuthMode.iamRole;
  }

  if (exporter?.awsAccessKey || exporter?.isAwsSecretKeySet) {
    return RdsAuthMode.accessKey;
  }

  return RdsAuthMode.hostCredentials;
};

const TLS_CERTIFICATES: SettingName[] = [
  'tlsCa',
  'tlsCert',
  'tlsKey',
  'tlsCertificateKey',
  'tlsCertificateKeyFilePassword',
];

const QAN_SETTINGS: SettingName[] = [
  'maxQueryLength',
  'disableQueryExamples',
  'disableCommentsParsing',
];

// Whether the service type has the setting and the form shows it, as
// certificates go with TLS and QAN settings with a query source
export const isSettingShown = (
  serviceType: ManagedServiceType,
  name: SettingName,
  { tls, querySource }: Pick<EditServiceFormValues, 'tls' | 'querySource'>
) => {
  if (!SERVICE_SETTINGS[serviceType].includes(name)) {
    return false;
  }

  if (TLS_CERTIFICATES.includes(name)) {
    return tls;
  }

  if (QAN_SETTINGS.includes(name)) {
    return querySource !== QuerySource.none;
  }

  if (name === 'maxSlowlogFileSize') {
    return querySource === QuerySource.slowlog;
  }

  return true;
};

const findAgent = (agents: ServiceAgent[], types: AgentType[]) =>
  agents.find(({ agentType }) => types.includes(agentType));

export const getQuerySource = (
  serviceType: ManagedServiceType,
  agents: ServiceAgent[]
) =>
  QUERY_SOURCES[serviceType]?.find(({ agentType }) =>
    agents.some((agent) => agent.agentType === agentType)
  )?.value ?? QuerySource.none;

// Settings a service shares across its agents are read from its exporter, as
// pmm-admin update does. The server refuses to update a service without one.
export const getExporter = ({ service, agents }: EditableService) => {
  const exporterType = EXPORTER_TYPES[service.serviceType];

  return (
    findAgent(agents, [exporterType]) ??
    toServiceAgent({ agent_id: '', agent_type: exporterType })
  );
};

// Without a QAN agent, the QAN settings are the ones the server gives a new one
export const getInitialValues = (
  editable: EditableService,
  rdsExporter: RdsExporter | null
): EditServiceFormValues => {
  const { service, agents } = editable;
  const { serviceType } = service;
  const exporter = getExporter(editable);
  const qan = findAgent(
    agents,
    (QUERY_SOURCES[serviceType] ?? []).map(({ agentType }) => agentType)
  );
  const slowlog = findAgent(agents, [AgentType.qanMysqlSlowlogAgent]);

  return {
    environment: service.environment,
    cluster: service.cluster,
    replicationSet: service.replicationSet,
    customLabels: toCustomLabelsText(service.customLabels),
    group: service.group,
    database: service.databaseName,
    username: exporter.username,
    // The API never returns secrets or certificates
    password: '',
    agentPassword: '',
    authenticationMechanism: exporter.authenticationMechanism,
    authenticationDatabase: exporter.authenticationDatabase,
    extraDsnParams: toCustomLabelsText(exporter.extraDsnParams),
    scheme: exporter.metricsScheme || 'http',
    metricsPath: exporter.metricsPath,
    listenPort: exporter.listenPort ? String(exporter.listenPort) : '',
    tls: exporter.tls,
    tlsSkipVerify: exporter.tlsSkipVerify,
    tlsCa: '',
    tlsCert: '',
    tlsKey: '',
    tlsCertificateKey: '',
    tlsCertificateKeyFilePassword: '',
    querySource: getQuerySource(serviceType, agents),
    maxQueryLength: String(qan?.maxQueryLength ?? 0),
    disableQueryExamples: qan?.queryExamplesDisabled ?? false,
    disableCommentsParsing: qan?.commentsParsingDisabled ?? true,
    maxSlowlogFileSize: toSlowlogFileSizeText(
      slowlog?.maxQueryLogSize ?? DEFAULT_SLOWLOG_FILE_SIZE
    ),
    metricsMode: exporter.pushMetrics ? MetricsMode.push : MetricsMode.pull,
    logLevel: exporter.logLevel,
    exposeExporter: exporter.exposeExporter,
    connectionTimeout: toSecondsText(exporter.connectionTimeout),
    disableCollectors: toListText(exporter.disabledCollectors),
    tablestatsGroupTableLimit: String(exporter.tablestatsGroupTableLimit),
    enableAllCollectors: exporter.enableAllCollectors,
    enableDiagnosticDataHistograms: exporter.enableDiagnosticDataHistograms,
    statsCollections: toListText(exporter.statsCollections),
    collectionsLimit: String(exporter.collectionsLimit),
    autoDiscoveryLimit: String(exporter.autoDiscoveryLimit),
    maxExporterConnections: String(exporter.maxExporterConnections),
    rdsAuthMode: getRdsAuthMode(rdsExporter),
    awsAccessKey: rdsExporter?.awsAccessKey ?? '',
    awsSecretKey: '',
    awsRoleArn: rdsExporter?.awsRoleArn ?? '',
  };
};

// The ChangeAgent fields for the chosen mode, or undefined when they are
// untouched. A role ARN is rejected next to an access key, so switching modes
// clears the abandoned fields (as `pmm-admin --aws-role-arn= ...` does).
export const toRdsCredentials = (
  values: EditServiceFormValues,
  exporter: RdsExporter | null
): RdsCredentials | undefined => {
  if (!exporter) {
    return undefined;
  }

  if (values.rdsAuthMode === RdsAuthMode.iamRole) {
    const awsRoleArn = values.awsRoleArn.trim();

    return awsRoleArn === exporter.awsRoleArn
      ? undefined
      : { awsRoleArn, awsAccessKey: '', awsSecretKey: '' };
  }

  if (values.rdsAuthMode === RdsAuthMode.accessKey) {
    const awsAccessKey = values.awsAccessKey.trim();
    const isUnchanged =
      awsAccessKey === exporter.awsAccessKey && !exporter.awsRoleArn;

    if (isUnchanged && !values.awsSecretKey) {
      return undefined;
    }

    // A blank secret key keeps the stored one
    return {
      awsAccessKey,
      awsRoleArn: '',
      ...(values.awsSecretKey && { awsSecretKey: values.awsSecretKey }),
    };
  }

  if (
    !exporter.awsAccessKey &&
    !exporter.awsRoleArn &&
    !exporter.isAwsSecretKeySet
  ) {
    return undefined;
  }

  return { awsAccessKey: '', awsSecretKey: '', awsRoleArn: '' };
};

const areLabelsEqual = (a: Record<string, string>, b: Record<string, string>) =>
  Object.keys(a).length === Object.keys(b).length &&
  Object.entries(a).every(([name, value]) => b[name] === value);

// Sent as typed; the secrets start blank, so a blank one keeps the stored value
const PLAIN_SETTINGS = [
  'environment',
  'cluster',
  'replicationSet',
  'group',
  'database',
  'username',
  'password',
  'agentPassword',
  'authenticationMechanism',
  'authenticationDatabase',
  'scheme',
  'tls',
  'tlsSkipVerify',
  'tlsCa',
  'tlsCert',
  'tlsKey',
  'tlsCertificateKey',
  'tlsCertificateKeyFilePassword',
  'disableQueryExamples',
  'disableCommentsParsing',
  'metricsMode',
  'logLevel',
  'exposeExporter',
  'enableAllCollectors',
  'enableDiagnosticDataHistograms',
] as const;

const LIST_SETTINGS = ['disableCollectors', 'statsCollections'] as const;

export const LABEL_CHANGES = new Set<string>([
  'environment',
  'cluster',
  'replicationSet',
  'customLabels',
  'group',
]);

// The UpdateService request for the settings that differ from the initial
// values, or undefined when none does
export const toServiceUpdate = (
  service: InventoryService,
  initial: EditServiceFormValues,
  values: EditServiceFormValues
): UpdateServiceParams | undefined => {
  const { serviceType } = service;
  const isEdited = (name: SettingName) =>
    isSettingShown(serviceType, name, values) && values[name] !== initial[name];
  const changes: Record<string, unknown> = {};

  for (const name of PLAIN_SETTINGS) {
    if (isEdited(name)) {
      changes[name] = values[name];
    }
  }

  for (const name of Object.keys(NUMBER_SETTINGS) as SettingName[]) {
    if (isEdited(name)) {
      changes[name] = Number(values[name]);
    }
  }

  for (const name of LIST_SETTINGS) {
    if (isEdited(name)) {
      changes[name] = parseList(values[name]);
    }
  }

  if (isEdited('extraDsnParams')) {
    changes.extraDsnParams = parseCustomLabels(values.extraDsnParams);
  }

  if (isEdited('metricsPath')) {
    changes.metricsPath = toMetricsPath(values.metricsPath.trim());
  }

  if (isEdited('maxSlowlogFileSize')) {
    changes.maxSlowlogFileSize = toSlowlogFileSize(values.maxSlowlogFileSize);
  }

  // Zero restores the default
  if (isEdited('connectionTimeout')) {
    changes.connectionTimeout = `${Number(values.connectionTimeout) || 0}s`;
  }

  if (isEdited('querySource')) {
    for (const { value, change } of QUERY_SOURCES[serviceType] ?? []) {
      changes[change] = values.querySource === value;
    }
  }

  // Labels left as they were are not parsed again, as a value can read like
  // another label (owner:DB Team:Lead)
  const customLabels =
    values.customLabels === initial.customLabels
      ? service.customLabels
      : parseCustomLabels(values.customLabels);

  if (!areLabelsEqual(customLabels, service.customLabels)) {
    changes.customLabels = customLabels;
  }

  if (!Object.keys(changes).length) {
    return undefined;
  }

  return {
    serviceId: service.serviceId,
    serviceType,
    changes: changes as ServiceChanges,
  };
};
