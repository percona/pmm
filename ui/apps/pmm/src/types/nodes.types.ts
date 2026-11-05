export enum NodeType {
  remoteRds = 'NODE_TYPE_REMOTE_RDS_NODE',
}

export interface RemoteRdsNode {
  nodeId: string;
  nodeName: string;
  address: string;
  nodeModel: string;
  region: string;
  az: string;
  customLabels: Record<string, string>;
  instanceId: string;
}

// Response from /v1/inventory/nodes API, which lists each Node type separately.
export interface ListNodesResponse {
  remoteRds?: RemoteRdsNode[];
}

export interface ListNodesParams {
  nodeType?: NodeType;
}
