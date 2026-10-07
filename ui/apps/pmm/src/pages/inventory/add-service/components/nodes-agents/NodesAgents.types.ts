import type { UniversalNode } from 'types/nodes.types';

export interface NodeOption extends Pick<
  UniversalNode,
  'nodeId' | 'nodeName' | 'isPmmServerNode'
> {
  agentIds: string[];
}
