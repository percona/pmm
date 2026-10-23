import { type AgentType, AgentStatus } from 'types/agent.types';
import { capitalize } from 'utils/text.utils';
import { Messages } from './Agents.messages';

export const getAgentStatusText = (status: AgentStatus) =>
  capitalize(
    status.replace('AGENT_STATUS_', '').replace(/_/g, ' ').toLowerCase()
  );

export const getAgentStatusColor = (status: AgentStatus) => {
  switch (status) {
    case AgentStatus.running:
    case AgentStatus.starting:
      return 'success';
    case AgentStatus.done:
      return 'default';
    default:
      return 'error';
  }
};

export const getAgentTypeName = (type: AgentType) =>
  Messages.agentTypes[type] ?? capitalize(type.replace(/[_-]/g, ' '));
