import { AgentStatus } from './agent.types';

export interface RunningRealTimeAgent {
  agentId: string;
  serviceId: string;
  serviceName: string;
  cluster: string;
  startedAt: Date;
  status: AgentStatus;
}

export interface ListRunningRealTimeAgentsResponse {
  agents: RunningRealTimeAgent[];
}

export interface ListRunningRealTimeAgentsRequest {
  cluster?: string;
}

export interface RealTimeSession {
  sessionId: string;
  type: 'service' | 'cluster';
  sessionName: string;
  status: AgentStatus;
  serviceSessions: RealTimeSession[];
}
