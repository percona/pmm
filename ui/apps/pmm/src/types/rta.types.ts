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
