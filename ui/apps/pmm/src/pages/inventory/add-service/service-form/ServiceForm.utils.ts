import { isAxiosError } from 'axios';
import { getApiErrorMessage } from 'api/api.utils';
import { InventoryNodeType } from 'types/nodes.types';
import { MetricsMode } from 'types/services.types';
import {
  LOCAL_ADDRESSES,
  MAX_CONNECTION_TIMEOUT_SECONDS,
} from './ServiceForm.constants';
import { Messages } from './ServiceForm.messages';
import type {
  CommonOptionsFormValues,
  ConnectionFormValues,
  DatabaseServiceFormValues,
  LabelsFormValues,
  TlsCertificatesFormValues,
} from './ServiceForm.types';

const LABEL_KEY_RE = /^[a-zA-Z_][a-zA-Z0-9_]*$/;
const COLLECTOR_RE = /^[a-zA-Z0-9_.-]+$/;
const DURATION_RE = /^-?\d+(?:\.\d+)?(ms|s|m)$/;

const splitKeyValues = (text: string) => text.split(/[\n\s]/).filter(Boolean);

const splitCollectors = (text: string) =>
  text
    .split(',')
    .map((collector) => collector.trim())
    .filter(Boolean);

export const isValidPort = (value: string) => {
  const port = Number(value);
  return Number.isInteger(port) && port > 0 && port <= 65535;
};

// Keys follow the Prometheus label name rules.
export const isValidKeyValues = (text: string) =>
  splitKeyValues(text).every((entry) => {
    const [key, value, ...rest] = entry.split(':');
    return (
      LABEL_KEY_RE.test(key) && !key.startsWith('__') && !!value && !rest.length
    );
  });

export const isValidCollectors = (text: string) =>
  splitCollectors(text).every((collector) => COLLECTOR_RE.test(collector));

export const getConnectionTimeoutError = (value: string) => {
  const timeout = value.trim();

  if (!timeout) {
    return undefined;
  }

  const unit = timeout.match(DURATION_RE)?.[1];

  if (!unit) {
    return Messages.validation.duration;
  }

  if (unit !== 's') {
    return Messages.validation.durationUnit;
  }

  const seconds = parseFloat(timeout);

  if (seconds < 0) {
    return Messages.validation.minDuration('0s');
  }

  if (seconds > MAX_CONNECTION_TIMEOUT_SECONDS) {
    return Messages.validation.maxDuration(
      `${MAX_CONNECTION_TIMEOUT_SECONDS}s`
    );
  }

  return undefined;
};

export const toKeyValueMap = (text: string) => {
  const entries = splitKeyValues(text);

  return entries.length
    ? Object.fromEntries(entries.map((entry) => entry.split(':')))
    : undefined;
};

export const toCollectors = (text: string) => {
  const collectors = splitCollectors(text);
  return collectors.length ? collectors : undefined;
};

export const toOptionalNumber = (value: string) =>
  value.trim() === '' ? undefined : Number(value);

export const toOptionalString = (value: string) => value.trim() || undefined;

export const getServiceName = ({
  serviceName,
  address,
}: Pick<ConnectionFormValues, 'serviceName' | 'address'>) =>
  serviceName.trim() || address.trim();

export const toLabels = ({
  environment,
  cluster,
  replicationSet,
  customLabels,
}: LabelsFormValues) => ({
  environment,
  cluster,
  replicationSet,
  customLabels: toKeyValueMap(customLabels),
});

export const toRemoteNode = (
  nodeName: string,
  { region, az }: Pick<LabelsFormValues, 'region' | 'az'>
) => ({
  nodeName,
  nodeType: InventoryNodeType.remote,
  region: toOptionalString(region),
  az: toOptionalString(az),
});

// A service on the agent's own host joins that node; anything else gets a remote node of its own.
export const toPlacement = (
  values: Pick<
    DatabaseServiceFormValues,
    'address' | 'nodeId' | 'region' | 'az'
  >,
  serviceName: string
) =>
  LOCAL_ADDRESSES.includes(values.address.trim()) && values.nodeId
    ? { nodeId: values.nodeId }
    : { addNode: toRemoteNode(serviceName, values) };

export const getMetricsMode = (isPmmServerNode: boolean) =>
  isPmmServerNode ? MetricsMode.pull : MetricsMode.push;

// Certificates stay in the form when TLS is turned off, so they are only sent with TLS on.
export const whenTls = <T extends object>(tls: boolean, certificates: T) =>
  tls ? certificates : {};

export const toTlsCertificates = ({
  tls,
  tlsCa,
  tlsCert,
  tlsKey,
}: Pick<DatabaseServiceFormValues, 'tls'> & TlsCertificatesFormValues) =>
  whenTls(tls, { tlsCa, tlsCert, tlsKey });

// A limit chosen by radio: the typed value for the editable option, a preset for the others.
export const toLimit = <T extends string>(
  option: T,
  editableOption: T,
  limit: string,
  presets: Partial<Record<T, number>>
) => (option === editableOption ? Number(limit) : presets[option]);

export const toRadioOptions = <T extends string>(
  values: T[],
  labels: Record<T, string>
) => values.map((value) => ({ value, label: labels[value] }));

// How PMM reaches a database, wherever it runs (remote, RDS or Azure).
export const toConnection = (values: DatabaseServiceFormValues) => ({
  serviceName: getServiceName(values),
  address: values.address.trim(),
  port: Number(values.port),
  pmmAgentId: values.pmmAgentId,
  username: values.username,
  password: values.password,
  connectionTimeout: toOptionalString(values.connectionTimeout),
  skipConnectionCheck: values.skipConnectionCheck,
  tls: values.tls,
  // Only offered with TLS on, so a choice made before turning TLS off is dropped.
  tlsSkipVerify: values.tls && values.tlsSkipVerify,
});

// Fields every database service (MySQL, PostgreSQL, MongoDB, ProxySQL, Valkey) is added with.
export const toDatabaseService = (values: DatabaseServiceFormValues) => {
  const connection = toConnection(values);

  return {
    ...connection,
    metricsMode: getMetricsMode(values.isPmmServerNode),
    ...toLabels(values),
    ...toPlacement(values, connection.serviceName),
  };
};

// Fields every exporter PMM scrapes over HTTP (HAProxy, External) is added with.
export const toScrapedService = (
  values: Pick<ConnectionFormValues, 'serviceName'> &
    LabelsFormValues &
    CommonOptionsFormValues,
  address: string
) => {
  const serviceName = getServiceName({
    serviceName: values.serviceName,
    address,
  });

  return {
    serviceName,
    address,
    ...toLabels(values),
    addNode: toRemoteNode(serviceName, values),
    metricsMode: MetricsMode.pull,
    skipConnectionCheck: values.skipConnectionCheck,
    tlsSkipVerify: values.tlsSkipVerify,
  };
};

export const getErrorMessage = (error: unknown) => {
  if (isAxiosError(error)) {
    return getApiErrorMessage(error, error.message);
  }

  return error instanceof Error ? error.message : Messages.unknownError;
};
