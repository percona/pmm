import { AgentUpdateSeverity, GetAgentVersionItem } from 'types/agent.types';

export const getAgentVersions = async (): Promise<GetAgentVersionItem[]> => [
  {
    agentId: 'pmm-server',
    version: '3.0.0',
    nodeName: 'pmm-server',
    severity: AgentUpdateSeverity.UP_TO_DATE,
  },
];
