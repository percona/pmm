import type { MRT_RowSelectionState } from '@percona/peak-ui';
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

// PMM Server refuses to remove its own nodes, and the nodes a PMM deployment
// provisioned for itself while their pmm-agent is connected
export const canRemoveNode = (node: ManagedNode) =>
  !node.isPmmServerNode && !node.isPmmProtectedNode;

// Returns the same selection unless it holds nodes that can't be removed
export const keepRemovableNodes = (
  selection: MRT_RowSelectionState,
  nodes: ManagedNode[]
): MRT_RowSelectionState => {
  const unremovable = new Set(
    nodes.filter((node) => !canRemoveNode(node)).map(({ nodeId }) => nodeId)
  );

  return Object.keys(selection).some((nodeId) => unremovable.has(nodeId))
    ? Object.fromEntries(
        Object.entries(selection).filter(([nodeId]) => !unremovable.has(nodeId))
      )
    : selection;
};
