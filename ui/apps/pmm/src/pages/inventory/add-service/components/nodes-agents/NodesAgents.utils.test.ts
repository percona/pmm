import { AgentStatus, ManagedAgentType } from 'types/agent.types';
import { ManagedNodeType, type UniversalNode } from 'types/nodes.types';
import { getPreselectedNode, toNodeOptions } from './NodesAgents.utils';

const node = (
  nodeId: string,
  agentTypes: ManagedAgentType[],
  overrides: Partial<UniversalNode> = {}
): UniversalNode => ({
  nodeId,
  nodeName: `${nodeId}-name`,
  nodeType: ManagedNodeType.generic,
  address: '',
  isPmmServerNode: false,
  isPmmInternalNode: false,
  agents: agentTypes.map((agentType, i) => ({
    agentId: `${nodeId}-agent-${i}`,
    agentType,
    status: AgentStatus.running,
    isConnected: true,
  })),
  ...overrides,
});

describe('NodesAgents.utils', () => {
  it('offers nodes that run a pmm-agent, with only their pmm-agents', () => {
    const options = toNodeOptions([
      node('client', [
        ManagedAgentType.pmmAgent,
        ManagedAgentType.nodeExporter,
      ]),
      node('exporter-only', [ManagedAgentType.nodeExporter]),
      node('internal', [ManagedAgentType.pmmAgent], {
        isPmmInternalNode: true,
      }),
    ]);

    expect(options).toEqual([
      {
        nodeId: 'client',
        nodeName: 'client-name',
        isPmmServerNode: false,
        agentIds: ['client-agent-0'],
      },
    ]);
  });

  it('preselects the PMM Server node, then the first node', () => {
    const client = {
      nodeId: 'client',
      nodeName: '',
      isPmmServerNode: false,
      agentIds: ['a'],
    };
    const server = { ...client, nodeId: 'pmm-server', isPmmServerNode: true };

    expect(getPreselectedNode([client, server])).toBe(server);
    expect(getPreselectedNode([client])).toBe(client);
    expect(getPreselectedNode([])).toBeUndefined();
  });
});
