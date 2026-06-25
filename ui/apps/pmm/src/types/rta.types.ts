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

export interface LockChainEntry {
  pid: number;
  lockMode: string;
  lockType: string;
  granted: boolean;
  queryText: string;
  duration?: string | null;
}

export interface QueryPostgreSQLData {
  pid: number;
  state: string;
  waitEventType: string;
  waitEvent: string;
  backendType: string;
  transactionStartTime?: string;
  stateChangeTime?: string;
  leaderPid?: number;
  lockChain?: LockChainEntry[];
  databaseName: string;
  username: string;
  applicationName: string;
  queryTruncated?: boolean;
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
  postgresqlPayload?: QueryPostgreSQLData;
}

export type QueryData = Exclude<RawQueryData, 'queryExecutionDuration'> & {
  queryExecutionDurationMs?: number | null;
  /** Transaction duration in seconds for idle-in-transaction sessions. */
  transactionDurationMs?: number | null;
  /** True when this row is a parallel worker collapsed under a leader. */
  isParallelWorker?: boolean;
  /** Leader query ID when isParallelWorker is true. */
  leaderQueryId?: string;
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

export interface AvailableServicesResponse {
  mongodb?: VersionedService[];
  postgresql?: VersionedService[];
}

export const isPostgreSQLQuery = (query: QueryData): boolean =>
  !!query.postgresqlPayload;

export const isIdleInTransaction = (query: QueryData): boolean =>
  query.postgresqlPayload?.state?.includes('idle in transaction') ?? false;

export const hasLockChain = (query: QueryData): boolean =>
  (query.postgresqlPayload?.lockChain?.length ?? 0) > 0;
