import { HighAvailabilityHealth } from './high-availability.types';

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
}

export interface GetHANodeResponse {
  nodeName: string;
  role: NodeRole;
  status: NodeStatus;
}

export interface HAInfo {
  enabled: boolean;
  health: HighAvailabilityHealth;
  leader?: GetHANodeResponse;
  nodes: GetHANodeResponse[];
}
