import { AgentStatus } from 'types/agent.types';
import { RealtimeSessionStatus } from 'types/rta.types';

export const getSessionStatusText = (status: RealtimeSessionStatus) => {
  switch (status) {
    case RealtimeSessionStatus.running:
      return 'Running';
    case RealtimeSessionStatus.error:
      return 'Error';
    case RealtimeSessionStatus.down:
      return 'Down';
    case RealtimeSessionStatus.unspecified:
      return 'Unspecified';
  }
};

export const getAgentStatusText = (status: AgentStatus) => {
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
