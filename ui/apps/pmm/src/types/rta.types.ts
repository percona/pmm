export enum RealtimeSessionStatus {
  unspecified = 'SESSION_STATUS_UNSPECIFIED',
  running = 'SESSION_STATUS_RUNNING',
  error = 'SESSION_STATUS_ERROR',
  down = 'SESSION_STATUS_DOWN',
}

export interface RealtimeSession {
  serviceId: string;
  serviceName: string;
  clusterName: string;
  startTime: string;
  status: RealtimeSessionStatus;
}

export interface ListRunningSessionsResponse {
  sessions: RealtimeSession[];
}

export interface StartSessionPayload {
  serviceId: string;
}

export interface StartSessionResponse {
  session: RealtimeSession;
}

export interface StopSessionPayload {
  serviceId: string;
}

export interface SearchQueriesPayload {
  serviceIds: string[];
  limit?: number;
}

export interface SearchQueriesResponse {
  queries: QueryData[];
}

export interface QueryData {
  serviceId: string;
  serviceName: string;
  queryId: string;
  queryText: string;
  state: string;
  executionDuration: string;
  rowsExamined: number;
  rowsSent: number;
  collectTime: string;
  rawQueryJson: string;
  mongoDbPayload?: QueryMongoDBData;
}

export interface QueryMongoDBData {
  opid: string;
  client: string;
  waitingForLock: boolean;
  indexUtilized: string;
}
