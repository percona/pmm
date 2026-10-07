import { NodeType, type UniversalNode } from 'types/nodes.types';
import { MetricsMode } from 'types/services.types';
import {
  LOCAL_ADDRESSES,
  MAX_CONNECTION_TIMEOUT_SECONDS,
} from './ServiceForm.constants';
import { Messages } from './ServiceForm.messages';
import type {
  ConnectionFormValues,
  LabelsFormValues,
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
  return Number.isInteger(port) && port > 0 && port < 65535;
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

// A service on the agent's own host joins that node; anything else gets a remote node of its own.
export const toPlacement = (
  address: string,
  serviceName: string,
  node?: Pick<UniversalNode, 'nodeId'>
) =>
  LOCAL_ADDRESSES.includes(address.trim()) && node
    ? { nodeId: node.nodeId }
    : { addNode: { nodeName: serviceName, nodeType: NodeType.remote } };

export const getMetricsMode = (node?: Pick<UniversalNode, 'isPmmServerNode'>) =>
  node?.isPmmServerNode ? MetricsMode.pull : MetricsMode.push;

// Certificates stay in the form when TLS is turned off, so they are only sent with TLS on.
export const whenTls = <T extends object>(tls: boolean, certificates: T) =>
  tls ? certificates : {};
