import type { AgentStatus, ManagedAgentType } from 'types/agent.types';

export enum NodeType {
  unspecified = 'NODE_TYPE_UNSPECIFIED',
  generic = 'NODE_TYPE_GENERIC_NODE',
  container = 'NODE_TYPE_CONTAINER_NODE',
  remote = 'NODE_TYPE_REMOTE_NODE',
  remoteRds = 'NODE_TYPE_REMOTE_RDS_NODE',
  remoteAzureDatabase = 'NODE_TYPE_REMOTE_AZURE_DATABASE_NODE',
}

// Node types returned by /v1/management/nodes API (lowercase format)
export enum ManagedNodeType {
  generic = 'generic',
  container = 'container',
  remote = 'remote',
  remoteRds = 'remote_rds',
  remoteAzureDatabase = 'remote_azure_database',
}

export interface NodeAgent {
  agentId: string;
  agentType: ManagedAgentType;
  status: AgentStatus;
  isConnected: boolean;
}

// Node from /v1/management/nodes API
export interface UniversalNode {
  nodeId: string;
  nodeType: ManagedNodeType;
  nodeName: string;
  address: string;
  agents: NodeAgent[];
  isPmmServerNode: boolean;
  isPmmInternalNode: boolean;
}

export interface ListNodesResponse {
  nodes: UniversalNode[];
}
