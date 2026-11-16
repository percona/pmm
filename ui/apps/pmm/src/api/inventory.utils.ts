import axios, {
  type AxiosRequestConfig,
  HttpStatusCode,
  isAxiosError,
} from 'axios';
import {
  type ManagedAgent,
  type RdsExporter,
  AgentStatus,
  AgentType,
} from 'types/agent.types';
import type { ManagedNode, NodeType } from 'types/nodes.types';
import type {
  BaseService,
  ManagedService,
  ManagedServiceType,
  ServiceStatus,
} from 'types/services.types';
import { api } from './api';

// Inventory payloads carry user-defined label keys, which the case middleware
// would rename, so they are parsed as is and mapped below.
export const RAW_RESPONSE: AxiosRequestConfig = {
  transformResponse: axios.defaults.transformResponse,
};

// Sends user-defined label keys as is, for the same reason
export const RAW_REQUEST: AxiosRequestConfig = {
  transformRequest: axios.defaults.transformRequest,
};

const isNotFound = (error: unknown) =>
  isAxiosError(error) && error.response?.status === HttpStatusCode.NotFound;

// Resolves to null when the entity no longer exists, for the page to report
// instead of the error notification
export const getIfExists = async <T>(url: string): Promise<T | null> => {
  try {
    const res = await api.get<T>(url, {
      ...RAW_RESPONSE,
      disableNotifications: isNotFound,
    });
    return res.data;
  } catch (error) {
    if (isNotFound(error)) {
      return null;
    }
    throw error;
  }
};

type PayloadValue = string | number | boolean | null | object | undefined;

interface AgentPayload {
  agent_id: string;
  agent_type: AgentType;
  status?: AgentStatus | '';
  is_connected?: boolean;
  disabled?: boolean;
  custom_labels?: Record<string, string>;
  aws_access_key?: string;
  aws_role_arn?: string;
  is_aws_secret_key_set?: boolean;
  [key: string]: PayloadValue;
}

export interface ServicePayload {
  service_id: string;
  service_type: ManagedServiceType;
  service_name: string;
  database_name: string;
  node_id: string;
  node_name: string;
  environment: string;
  cluster: string;
  replication_set: string;
  custom_labels?: Record<string, string>;
  external_group: string;
  address: string;
  port: number;
  socket: string;
  version: string;
  status: ServiceStatus;
  agents?: AgentPayload[];
  [key: string]: PayloadValue;
}

export interface NodePayload {
  node_id: string;
  node_type: NodeType;
  node_name: string;
  address: string;
  status: ServiceStatus;
  custom_labels?: Record<string, string>;
  is_pmm_server_node: boolean;
  agents?: AgentPayload[];
  services?: {
    service_id: string;
    service_type: ManagedServiceType;
    service_name: string;
  }[];
  [key: string]: PayloadValue;
}

export interface ListAgentsPayload {
  agents?: AgentPayload[];
}

export interface ListNodesPayload {
  nodes?: NodePayload[];
}

export interface ListServicesPayload {
  services?: ServicePayload[];
}

interface InventoryServicePayload {
  service_id: string;
  service_name: string;
  node_id: string;
  environment: string;
  cluster: string;
  replication_set: string;
  custom_labels?: Record<string, string>;
}

// The service is nested under its type, e.g. { mysql: { ... } }
export type GetServicePayload = Record<string, InventoryServicePayload>;

export interface GetNodePayload {
  node: NodePayload;
}

const SERVICE_MAIN_FIELDS = [
  'service_id',
  'service_type',
  'service_name',
  'custom_labels',
  'node_id',
  'address',
  'port',
  'agents',
  'node_name',
  'status',
];

const NODE_MAIN_FIELDS = [
  'address',
  'services',
  'agents',
  'node_type',
  'node_id',
  'node_name',
  'status',
  'custom_labels',
];

const AGENT_MAIN_FIELDS = [
  'agent_id',
  'agent_type',
  'node_id',
  'custom_labels',
  'status',
  'is_connected',
  'disabled',
];

// Nested agent options are flattened without their parent key, e.g.
// mysql_options.is_tls_key_set becomes is_tls_key_set.
const AGENT_UNPREFIXED_OPTIONS = [
  'azure_options',
  'mongo_db_options',
  'mysql_options',
  'postgresql_options',
  'valkey_options',
];

const getScalarFields = (
  payload: Record<string, PayloadValue>,
  mainFields: string[]
) => {
  const fields: Record<string, string> = {};

  for (const [key, value] of Object.entries(payload)) {
    const isScalar =
      value !== null && (typeof value !== 'object' || Array.isArray(value));

    if (!mainFields.includes(key) && isScalar && value !== '') {
      fields[key] = String(value);
    }
  }

  return fields;
};

const getAgentParams = (payload: AgentPayload) => {
  const params: Record<string, string> = {};

  const addParam = (key: string, value: PayloadValue) => {
    if (!value) {
      return;
    }

    if (Array.isArray(value)) {
      params[key] = value.join(',');
    } else if (typeof value === 'object') {
      for (const [childKey, childValue] of Object.entries(value)) {
        addParam(
          AGENT_UNPREFIXED_OPTIONS.includes(key)
            ? childKey
            : `${key}.${childKey}`,
          childValue
        );
      }
    } else {
      params[key] = String(value);
    }
  };

  for (const [key, value] of Object.entries(payload)) {
    if (!AGENT_MAIN_FIELDS.includes(key)) {
      addParam(key, value);
    }
  }

  return params;
};

const getAgentStatus = (payload: AgentPayload) => {
  if (payload.status) {
    return payload.status;
  }

  return payload.is_connected ? AgentStatus.running : AgentStatus.unknown;
};

export const toManagedAgent = (payload: AgentPayload): ManagedAgent => ({
  agentId: payload.agent_id,
  agentType: payload.agent_type,
  status: getAgentStatus(payload),
  isConnected: !!payload.is_connected,
  disabled: !!payload.disabled,
  properties: { ...payload.custom_labels, ...getAgentParams(payload) },
});

export const toManagedService = (payload: ServicePayload): ManagedService => ({
  serviceId: payload.service_id,
  serviceType: payload.service_type,
  serviceName: payload.service_name,
  databaseName: payload.database_name,
  nodeId: payload.node_id,
  nodeName: payload.node_name,
  environment: payload.environment,
  cluster: payload.cluster,
  replicationSet: payload.replication_set,
  customLabels: payload.custom_labels ?? {},
  externalGroup: payload.external_group,
  address: payload.address,
  port: payload.port,
  socket: payload.socket,
  version: payload.version,
  status: payload.status,
  agents: (payload.agents ?? []).map(toManagedAgent),
  labels: {
    ...payload.custom_labels,
    ...getScalarFields(payload, SERVICE_MAIN_FIELDS),
  },
});

export const toBaseService = (
  payload: InventoryServicePayload
): BaseService => ({
  serviceId: payload.service_id,
  serviceName: payload.service_name,
  nodeId: payload.node_id,
  environment: payload.environment,
  cluster: payload.cluster,
  replicationSet: payload.replication_set,
  customLabels: payload.custom_labels ?? {},
});

export const toManagedNode = (payload: NodePayload): ManagedNode => ({
  nodeId: payload.node_id,
  nodeType: payload.node_type,
  nodeName: payload.node_name,
  address: payload.address,
  status: payload.status,
  customLabels: payload.custom_labels ?? {},
  isPmmServerNode: payload.is_pmm_server_node,
  agents: (payload.agents ?? []).map(toManagedAgent),
  services: (payload.services ?? []).map((service) => ({
    serviceId: service.service_id,
    serviceType: service.service_type,
    serviceName: service.service_name,
  })),
  properties: getScalarFields(payload, NODE_MAIN_FIELDS),
});

export const toRdsExporter = (payload: AgentPayload): RdsExporter => ({
  agentId: payload.agent_id,
  awsAccessKey: payload.aws_access_key ?? '',
  awsRoleArn: payload.aws_role_arn ?? '',
  isAwsSecretKeySet: !!payload.is_aws_secret_key_set,
});

// The exporter is looked up by node, so credentials are only editable when
// there is no doubt which one they belong to
export const findRdsExporter = (agents: AgentPayload[] = []) => {
  const exporters = agents.filter(
    ({ agent_type }) => agent_type === AgentType.rdsExporter
  );

  return exporters.length === 1 ? toRdsExporter(exporters[0]) : null;
};
