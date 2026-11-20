import type { GetHANodeResponse } from 'types/ha.types';
import type { ManagedNode } from 'types/nodes.types';
import { getAgentsMonitoringStatus } from '../Inventory.utils';
import type { NodeRow } from './Nodes.types';

export const toNodeRows = (
  nodes: ManagedNode[],
  haNodes: GetHANodeResponse[]
): NodeRow[] =>
  nodes
    .map((node) => ({
      ...node,
      agentsStatus: getAgentsMonitoringStatus(node.agents),
      haRole: haNodes.find(({ nodeName }) => nodeName === node.nodeName)?.role,
    }))
    .sort((a, b) => a.nodeName.localeCompare(b.nodeName));

// PMM Server refuses to remove its own nodes
export const canRemoveNode = (node: ManagedNode) => !node.isPmmServerNode;
