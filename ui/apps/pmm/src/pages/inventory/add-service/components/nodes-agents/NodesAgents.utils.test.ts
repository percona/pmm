import { AgentStatus, AgentType } from 'types/agent.types';
import { NodeType, type ManagedNode } from 'types/nodes.types';
import { ServiceStatus } from 'types/services.types';
import { getPreselectedNode, toNodeOptions } from './NodesAgents.utils';

const node = (
  nodeId: string,
  agentTypes: AgentType[],
  overrides: Partial<ManagedNode> = {}
): ManagedNode => ({
  nodeId,
  nodeName: `${nodeId}-name`,
  nodeType: NodeType.generic,
  address: '',
  isPmmServerNode: false,
  isPmmInternalNode: false,
  status: ServiceStatus.up,
  customLabels: {},
  services: [],
  properties: {},
  agents: agentTypes.map((agentType, i) => ({
    agentId: `${nodeId}-agent-${i}`,
    agentType,
    status: AgentStatus.running,
    isConnected: true,
    disabled: false,
    properties: {},
  })),
  ...overrides,
});

describe('NodesAgents.utils', () => {
  it('offers nodes that run a pmm-agent, with only their pmm-agents', () => {
    const options = toNodeOptions([
      node('client', [AgentType.pmmAgent, AgentType.nodeExporter]),
      node('exporter-only', [AgentType.nodeExporter]),
      node('internal', [AgentType.pmmAgent], {
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
