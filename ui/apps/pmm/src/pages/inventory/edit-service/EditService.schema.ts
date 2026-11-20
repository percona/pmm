import { LogLevel, type RdsExporter } from 'types/agent.types';
import { ManagedServiceType, MetricsMode } from 'types/services.types';
import { z } from 'zod';
import { MAX_PORT, NUMBER_SETTINGS } from './EditService.constants';
import { Messages } from './EditService.messages';
import {
  QuerySource,
  RdsAuthMode,
  type SettingName,
} from './EditService.types';
import {
  isSettingShown,
  splitCustomLabel,
  splitCustomLabelEntries,
} from './EditService.utils';

// Prometheus label names, see https://prometheus.io/docs/concepts/data_model
const LABEL_NAME_RE = /^[a-zA-Z_][a-zA-Z0-9_]*$/;
const DSN_PARAM_NAME_RE = /^\w+$/;
// Mirrors the server's rule on aws_role_arn in api/inventory/v1/agents.proto
const AWS_ROLE_ARN_RE = /^arn:aws[a-zA-Z0-9-]*:iam::[0-9]{12}:role\/.+$/;
const WHOLE_NUMBER_RE = /^-?\d+$/;

// The API requires a username for these
const USERNAME_REQUIRED_TYPES = [
  ManagedServiceType.mysql,
  ManagedServiceType.postgresql,
  ManagedServiceType.proxysql,
];

const arePairsValid = (text: string, isNameValid: (name: string) => boolean) =>
  splitCustomLabelEntries(text).every((entry) => {
    const [name, value] = splitCustomLabel(entry);

    return isNameValid(name) && !!value;
  });

export const areCustomLabelsValid = (text: string) =>
  arePairsValid(
    text,
    (name) => LABEL_NAME_RE.test(name) && !name.startsWith('__')
  );

export const areDsnParamsValid = (text: string) =>
  arePairsValid(text, (name) => DSN_PARAM_NAME_RE.test(name));

const isWholeNumber = (text: string, min: number, max = Infinity) =>
  WHOLE_NUMBER_RE.test(text.trim()) &&
  Number(text) >= min &&
  Number(text) <= max;

const editServiceFields = z.object({
  environment: z.string(),
  cluster: z.string(),
  replicationSet: z.string(),
  customLabels: z.string(),
  group: z.string(),
  database: z.string(),
  username: z.string(),
  password: z.string(),
  agentPassword: z.string(),
  authenticationMechanism: z.string(),
  authenticationDatabase: z.string(),
  extraDsnParams: z.string(),
  scheme: z.string(),
  metricsPath: z.string(),
  listenPort: z.string(),
  tls: z.boolean(),
  tlsSkipVerify: z.boolean(),
  tlsCa: z.string(),
  tlsCert: z.string(),
  tlsKey: z.string(),
  tlsCertificateKey: z.string(),
  tlsCertificateKeyFilePassword: z.string(),
  querySource: z.enum(QuerySource),
  maxQueryLength: z.string(),
  disableQueryExamples: z.boolean(),
  disableCommentsParsing: z.boolean(),
  maxSlowlogFileSize: z.string(),
  metricsMode: z.enum(MetricsMode),
  logLevel: z.enum(LogLevel),
  exposeExporter: z.boolean(),
  connectionTimeout: z.string(),
  disableCollectors: z.string(),
  tablestatsGroupTableLimit: z.string(),
  enableAllCollectors: z.boolean(),
  enableDiagnosticDataHistograms: z.boolean(),
  statsCollections: z.string(),
  collectionsLimit: z.string(),
  autoDiscoveryLimit: z.string(),
  maxExporterConnections: z.string(),
  rdsAuthMode: z.enum(RdsAuthMode),
  awsAccessKey: z.string(),
  awsSecretKey: z.string(),
  awsRoleArn: z.string(),
});

export type EditServiceFormValues = z.infer<typeof editServiceFields>;

type FieldErrors = [keyof EditServiceFormValues, string][];

// Stored values are only checked once edited, as the server may hold values
// these rules would reject
const getSettingsErrors = (
  serviceType: ManagedServiceType,
  initial: EditServiceFormValues,
  values: EditServiceFormValues
) => {
  const { validation } = Messages;
  const errors: FieldErrors = [];
  const isEdited = (name: SettingName) =>
    isSettingShown(serviceType, name, values) && values[name] !== initial[name];

  if (isEdited('customLabels') && !areCustomLabelsValid(values.customLabels)) {
    errors.push(['customLabels', Messages.labels.customLabelsInvalid]);
  }

  if (isEdited('extraDsnParams') && !areDsnParamsValid(values.extraDsnParams)) {
    errors.push(['extraDsnParams', validation.dsnParamsInvalid]);
  }

  if (
    isEdited('username') &&
    USERNAME_REQUIRED_TYPES.includes(serviceType) &&
    !values.username.trim()
  ) {
    errors.push(['username', validation.required]);
  }

  for (const [name, min] of Object.entries(NUMBER_SETTINGS)) {
    const setting = name as SettingName;
    const max = setting === 'listenPort' ? MAX_PORT : Infinity;

    if (
      isEdited(setting) &&
      !isWholeNumber(String(values[setting]), min, max)
    ) {
      errors.push([
        setting,
        setting === 'listenPort'
          ? validation.invalidPort
          : validation.invalidNumber(min),
      ]);
    }
  }

  if (
    isEdited('maxSlowlogFileSize') &&
    !isWholeNumber(values.maxSlowlogFileSize, -1)
  ) {
    errors.push(['maxSlowlogFileSize', validation.invalidNumber(-1)]);
  }

  const timeout = values.connectionTimeout.trim();

  if (isEdited('connectionTimeout') && timeout && !(Number(timeout) >= 0)) {
    errors.push(['connectionTimeout', validation.invalidTimeout]);
  }

  return errors;
};

const getRdsCredentialsErrors = (
  values: EditServiceFormValues,
  exporter: RdsExporter
) => {
  const { required, secretKeyRequired, invalidRoleArn } = Messages.rds;
  const errors: FieldErrors = [];

  if (values.rdsAuthMode === RdsAuthMode.accessKey) {
    const accessKey = values.awsAccessKey.trim();

    if (!accessKey) {
      errors.push(['awsAccessKey', required]);
    }

    // A stored secret key only goes with the stored access key
    if (
      !values.awsSecretKey &&
      !(exporter.isAwsSecretKeySet && accessKey === exporter.awsAccessKey)
    ) {
      errors.push(['awsSecretKey', secretKeyRequired]);
    }
  }

  if (values.rdsAuthMode === RdsAuthMode.iamRole) {
    const roleArn = values.awsRoleArn.trim();

    if (!roleArn) {
      errors.push(['awsRoleArn', required]);
    } else if (!AWS_ROLE_ARN_RE.test(roleArn)) {
      errors.push(['awsRoleArn', invalidRoleArn]);
    }
  }

  return errors;
};

export const getEditServiceSchema = (
  serviceType: ManagedServiceType,
  initial: EditServiceFormValues,
  exporter: RdsExporter | null
) =>
  editServiceFields.superRefine((values, ctx) => {
    const errors = [
      ...getSettingsErrors(serviceType, initial, values),
      ...(exporter ? getRdsCredentialsErrors(values, exporter) : []),
    ];

    for (const [path, message] of errors) {
      ctx.addIssue({ code: 'custom', path: [path], message });
    }
  });
