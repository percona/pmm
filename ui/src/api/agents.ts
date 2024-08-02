import {
  AgentUpdateSeverity,
  GetAgentVersionsResponse,
} from 'types/agent.types';
import { api } from './api';

export const getAgentVersions = async () => {
  const res = await api.get<GetAgentVersionsResponse>(
    '/management/agents/versions'
  );
  const agent = res.data.agentVersions[0];
  return [
    agent,
    {
      ...agent,
      agentId: '842d31b3-8380-4b4c-839c-be56e7444d0f',
      nodeName: 'Critical',
      severity: AgentUpdateSeverity.CRITICAL,
    },
    {
      ...agent,
      agentId: '15a75c7f-2ca6-4cb6-b410-e6b86bb42b53',
      nodeName: 'Required',
      severity: AgentUpdateSeverity.REQUIRED,
    },
    {
      ...agent,
      agentId: '96acbad0-9d96-41cb-ab28-5e8fe50cfd52',
      nodeName: 'Unsupported',
      severity: AgentUpdateSeverity.UNSUPPORTED,
    },
  ];
};
