import { AgentStatus } from 'types/agent.types';

export const getStatusText = (status: AgentStatus) => {
  switch (status) {
    case AgentStatus.RUNNING:
      return 'Running';
    case AgentStatus.STARTING:
      return 'Starting';
    case AgentStatus.INITIALIZATION_ERROR:
      return 'Initialization error';
    case AgentStatus.WAITING:
      return 'Waiting';
    case AgentStatus.STOPPING:
      return 'Stopping';
    case AgentStatus.DONE:
      return 'Done';
    case AgentStatus.UNKNOWN:
      return 'Unknown';
    case AgentStatus.UNSPECIFIED:
      return 'Unspecified';
  }
};
