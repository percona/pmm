export enum AgentStatus {
  Unknown = 'AGENT_STATUS_UNKNOWN',
  Starting = 'AGENT_STATUS_STARTING',
  Running = 'AGENT_STATUS_RUNNING',
  Waiting = 'AGENT_STATUS_WAITING',
  Stopping = 'AGENT_STATUS_STOPPING',
  Done = 'AGENT_STATUS_DONE',
}

export interface RunningRealtimeAgent {
  agentId: string;
  serviceId: string;
  serviceName: string;
  cluster: string;
  startedAt: string;
  status: AgentStatus;
}

export interface ListRunningRealtimeAgentsRequest {
  cluster?: string;
}

export interface ListRunningRealtimeAgentsResponse {
  agents: RunningRealtimeAgent[];
}

export interface ChangeRealtimeAnalyticsRequest {
  enable: boolean;
  serviceId: string;
}

export interface ChangeRealtimeAnalyticsResponse {
  // Empty response
}
