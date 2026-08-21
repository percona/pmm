import { ServiceType, VersionedService } from './services.types';

export enum RealtimeSessionStatus {
  unspecified = 'SESSION_STATUS_UNSPECIFIED',
  running = 'SESSION_STATUS_RUNNING',
  error = 'SESSION_STATUS_ERROR',
  down = 'SESSION_STATUS_DOWN',
}

export interface RealtimeSession {
  serviceId: string;
  serviceName: string;
  // Database technology of the monitored service, so MySQL and MongoDB
  // sessions can be told apart without a second inventory lookup.
  serviceType: ServiceType;
  clusterName: string;
  startTime: string;
  status: RealtimeSessionStatus;
}

// A service that can have an RTA session started for it, carrying the
// technology it was listed under.
export type AvailableService = VersionedService & {
  serviceType: ServiceType;
};

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
  queries: RawQueryData[];
}

export interface RawQueryData {
  serviceId: string;
  serviceName: string;
  queryId: string;
  queryText: string;
  queryExecutionDuration?: string | null;
  queryCollectTime: string;
  clientAddress: string;
  queryRawJson: string;
  // Exactly one of the payloads below is set depending on the database type.
  mongoDbPayload?: QueryMongoDBData;
  mySqlPayload?: QueryMySQLData;
}

export type QueryData = Omit<RawQueryData, 'queryExecutionDuration'> & {
  queryExecutionDurationMs?: number | null;
};

export interface QueryMongoDBData {
  dbInstanceAddress: string;
  clientAppName: string;
  databaseName: string;
  operationStartTime: string;
  planSummary: string;
  operation: string;
  username: string;
  collection?: string;
}

export interface QueryMySQLData {
  dbInstanceAddress: string;
  programName: string;
  databaseName: string;
  command: string;
  state: string;
  username: string;
  rowsExamined?: number | string;
  rowsSent?: number | string;
  fullScan?: boolean;
}

// TODO: Add other service types when available
export interface AvailableServicesResponse {
  mongodb?: VersionedService[];
  mysql?: VersionedService[];
}
