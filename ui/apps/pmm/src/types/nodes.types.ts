import type { ManagedAgent } from './agent.types';
import type { ManagedServiceType, ServiceStatus } from './services.types';

export enum NodeType {
  generic = 'generic',
  container = 'container',
  remote = 'remote',
  remoteRds = 'remote_rds',
  remoteAzureDatabase = 'remote_azure_database',
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
  agents: ManagedAgent[];
  services: NodeService[];
  // The node's other scalar attributes, keyed as the API names them
  properties: Record<string, string>;
}

export interface RemoveNodeParams {
  nodeId: string;
  force: boolean;
}
