import { AgentType } from 'types/agent.types';
import type { ManagedNode } from 'types/nodes.types';
import type { NodeOption } from './NodesAgents.types';

// Internal nodes (e.g. the HA persistence layer) must not take monitoring workloads.
export const toNodeOptions = (nodes: ManagedNode[]): NodeOption[] =>
  nodes
    .filter((node) => !node.isPmmInternalNode)
    .map(({ nodeId, nodeName, isPmmServerNode, agents }) => ({
      nodeId,
      nodeName,
      isPmmServerNode,
      agentIds: agents
        .filter((agent) => agent.agentType === AgentType.pmmAgent)
        .map((agent) => agent.agentId),
    }))
    .filter((node) => node.agentIds.length > 0);

export const getPreselectedNode = (nodes: NodeOption[]) =>
  nodes.find((node) => node.isPmmServerNode) ?? nodes[0];
