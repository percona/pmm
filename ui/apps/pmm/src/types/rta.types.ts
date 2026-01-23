import { AgentStatus } from './agent.types';

export interface RunningRealTimeAgent {
  agentId: string;
  serviceId: string;
  serviceName: string;
  cluster: string;
  startedAt: string;
  status: AgentStatus;
}

export interface ListRunningRealTimeAgentsResponse {
  agents: RunningRealTimeAgent[];
}

export interface ListRunningRealTimeAgentsRequest {
  cluster?: string;
}

// export interface RealTimeSession {
//   sessionId: string;
//   type: 'service' | 'cluster';
//   sessionName: string;
//   status: AgentStatus;
//   serviceSessions: RealTimeSession[];
//   agents: RunningRealTimeAgent[];
//   startedAt: string;
// }

export interface ChangeRealTimeAgentPayload {
  serviceId: string;
  enable: boolean;
}

export interface ChangeRealTimeAgentResponse { }

export enum RealTimeSessionStatus {
  unspecified = 'SESSION_STATUS_UNSPECIFIED',
  running = 'SESSION_STATUS_RUNNING',
  error = 'SESSION_STATUS_ERROR',
  down = 'SESSION_STATUS_DOWN',
}

export interface RealTimeSession {
  serviceId: string;
  serviceName: string;
  clusterName: string;
  startTime: string;
  status: RealTimeSessionStatus;
}

export interface ListRunningSessionsResponse {
  sessions: RealTimeSession[];
}

export interface StartSessionPayload {
  serviceId: string;
}

export interface StartSessionResponse {
  session: RealTimeSession;
}

export interface StopSessionPayload {
  serviceId: string;
}

export interface StopSessionResponse {
  // empty response
}