import type { ManagedNode } from 'types/nodes.types';

export interface NodeOption extends Pick<
  ManagedNode,
  'nodeId' | 'nodeName' | 'isPmmServerNode'
> {
  agentIds: string[];
}
