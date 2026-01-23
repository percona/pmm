import { AgentStatus } from 'types/agent.types';
import { RealTimeSessionStatus } from "types/rta.types";

export const getSessionStatusText = (status: RealTimeSessionStatus) => {
    switch (status) {
        case RealTimeSessionStatus.running:
            return 'Running';
        case RealTimeSessionStatus.error:
            return 'Error';
        case RealTimeSessionStatus.down:
            return 'Down';
        case RealTimeSessionStatus.unspecified:
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
