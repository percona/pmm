import { VersionedService } from './services.types';

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
  queries: RawQueryData[];
}

export interface LockChainLink {
  blockerPid: number;
  blockedPid: number;
  lockMode: string;
  relationName: string;
  blockerQueryText: string;
  blockerDuration?: string | null;
}

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

export interface QueryPostgreSQLData {
  dbInstanceAddress: string;
  databaseName: string;
  username: string;
  applicationName: string;
  sessionState: string;
  transactionStartTime?: string;
  queryStartTime?: string;
  waitEventType: string;
  waitEvent: string;
  backendPid: number;
  leaderPid: number;
  queryTextTruncated: boolean;
  trackActivityQuerySize: number;
  lockChain?: LockChainLink[];
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
  mongoDbPayload?: QueryMongoDBData;
  postgresPayload?: QueryPostgreSQLData;
}

export type QueryData = Exclude<RawQueryData, 'queryExecutionDuration'> & {
  queryExecutionDurationMs?: number | null;
};

export interface AvailableServicesResponse {
  mongodb: VersionedService[];
  postgresql: VersionedService[];
}

export const isPostgresQuery = (query: QueryData): boolean =>
  !!query.postgresPayload && Object.keys(query.postgresPayload).length > 0;

export const isMongoQuery = (query: QueryData): boolean =>
  !!query.mongoDbPayload && Object.keys(query.mongoDbPayload).length > 0;
