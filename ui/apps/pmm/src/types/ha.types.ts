export interface GetHAStatusResponse {
  status: HAStatus;
}

export type HAStatus = 'Enabled' | 'Disabled';

export enum NodeRole {
  leader = 'NODE_ROLE_LEADER',
  follower = 'NODE_ROLE_FOLLOWER',
  unspecified = 'NODE_ROLE_UNSPECIFIED',
}

export type NodeStatus = 'alive' | 'suspect' | 'dead' | 'left' | 'unknown';

export interface GetHANodesResponse {
  nodes: GetHANodeResponse[];
  expectedNodes: number;
}

export interface GetHANodeResponse {
  nodeName: string;
  role: NodeRole;
  status: NodeStatus;
}

export interface HAInfo {
  enabled: boolean;
  health: HAHealth;
  leader?: GetHANodeResponse;
  nodes: GetHANodeResponse[];
}

export type HAHealth =
  | 'healthy'
  | 'degraded'
  | 'critical'
  | 'unreachable'
  | 'unknown';
