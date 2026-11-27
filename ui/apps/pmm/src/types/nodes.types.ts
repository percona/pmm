import type { ManagedAgent } from './agent.types';
import type { ManagedServiceType, ServiceStatus } from './services.types';

export enum NodeType {
  generic = 'generic',
  container = 'container',
  remote = 'remote',
  remoteRds = 'remote_rds',
  remoteAzureDatabase = 'remote_azure_database',
}

// Node types the inventory API takes, e.g. for the node a new service adds
export enum InventoryNodeType {
  unspecified = 'NODE_TYPE_UNSPECIFIED',
  generic = 'NODE_TYPE_GENERIC_NODE',
  container = 'NODE_TYPE_CONTAINER_NODE',
  remote = 'NODE_TYPE_REMOTE_NODE',
  remoteRds = 'NODE_TYPE_REMOTE_RDS_NODE',
  remoteAzureDatabase = 'NODE_TYPE_REMOTE_AZURE_DATABASE_NODE',
}

export interface NodeService {
  serviceId: string;
  serviceType: ManagedServiceType;
  serviceName: string;
}

// Node from /v1/management/nodes API
export interface ManagedNode {
  nodeId: string;
  nodeType: NodeType;
  nodeName: string;
  address: string;
  // Nodes report the same statuses as services
  status: ServiceStatus;
  customLabels: Record<string, string>;
  isPmmServerNode: boolean;
  // Nodes PMM Server runs for itself; the add-service node picker leaves them out
  isPmmInternalNode?: boolean;
  agents: ManagedAgent[];
  services: NodeService[];
  // The node's other scalar attributes, keyed as the API names them
  properties: Record<string, string>;
}

export interface RemoveNodeParams {
  nodeId: string;
  force: boolean;
}

export interface RemoveNodeResponse {
  // Names what the removal left behind, e.g. the node's service account
  warning?: string;
}

export interface RemoveNodesResult {
  deleted: number;
  warnings: string[];
}
