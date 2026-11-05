export enum NodeType {
  generic = 'NODE_TYPE_GENERIC_NODE',
  container = 'NODE_TYPE_CONTAINER_NODE',
  remote = 'NODE_TYPE_REMOTE_NODE',
  remoteRds = 'NODE_TYPE_REMOTE_RDS_NODE',
  remoteAzureDatabase = 'NODE_TYPE_REMOTE_AZURE_DATABASE_NODE',
}

// Response from /v1/inventory/nodes:getTypes, the Node types in the inventory.
export interface ListNodeTypesResponse {
  nodeTypes?: NodeType[];
}
