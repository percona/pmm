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
  serviceType: ServiceType;
  clusterName: string;
  startTime: string;
  status: RealtimeSessionStatus;
  // Why the session failed to start, or what it cannot collect; never sent by older agents.
  statusMessage?: string;
}

// A service RTA can start a session for, with the technology it was listed under.
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
  postgresqlPayload?: QueryPostgreSQLData;
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
  // UNSPECIFIED means the lock graph was unreadable and must not be shown as "not blocked".
  blockedStatus?: BlockedStatus;
  blockedBy?: BlockingTransaction[];
  // A property of the waiter: every blocker of a statement contends for the same lock.
  lockedTable?: string;
  // Row locks only; a metadata lock covers the whole table.
  lockedIndex?: string;
  lockType?: LockType;
  requestedLockMode?: string;
  // MySQL keeps only the beginning of a long statement and marks no cut of its own.
  queryTextTruncated?: boolean;
  // Protobuf duration ("0.000003s"); absent when the server did not measure it.
  lockTime?: string | null;
}

// Read from pg_stat_activity; blockers come from pg_blocking_pids(), with the
// blocker's pid as blockingConnId and its session state as blockingCommand.
export interface QueryPostgreSQLData {
  dbInstanceAddress?: string;
  databaseName: string;
  username: string;
  applicationName: string;
  state: string;
  waitEventType: string;
  waitEvent: string;
  pid: number;
  // PostgreSQL 14+ with compute_query_id; empty otherwise.
  queryId: string;
  transactionStartTime?: string;
  queryStartTime?: string;
  // Cut at track_activity_query_size.
  queryTextTruncated?: boolean;
  blockedStatus?: BlockedStatus;
  blockedBy?: BlockingTransaction[];
}

export enum BlockedStatus {
  unspecified = 'BLOCKED_STATUS_UNSPECIFIED',
  notBlocked = 'BLOCKED_STATUS_NOT_BLOCKED',
  blocked = 'BLOCKED_STATUS_BLOCKED',
  // The wait belonged to a later statement on the connection; unlike unspecified, a refresh
  // rather than a configuration change resolves it.
  unattributed = 'BLOCKED_STATUS_UNATTRIBUTED',
}

// A row lock is freed when the holding transaction ends, a metadata lock when the holder
// finishes its statement or transaction on that table.
export enum LockType {
  unspecified = 'LOCK_TYPE_UNSPECIFIED',
  row = 'LOCK_TYPE_ROW',
  metadata = 'LOCK_TYPE_METADATA',
}

// A transaction preventing a statement from taking the lock it is waiting for.
export interface BlockingTransaction {
  blockingConnId: number | string;
  blockingQuery: string;
  // "Sleep" means the blocker is idle in an open transaction, so blockingQuery is its last
  // statement, which need not be the one that took the lock.
  blockingCommand: string;
  blockingUsername: string;
  waitDuration?: string | null;
  blockerTransactionDuration?: string | null;
  // Not itself waiting, so resolving it releases everything queued behind it.
  root?: boolean;
  // E.g. "X,REC_NOT_GAP" for a row lock, "SHARED_READ" for a metadata lock.
  blockingLockMode?: string;
  // blockingQuery is incomplete; see QueryMySQLData.queryTextTruncated.
  blockingQueryTruncated?: boolean;
}

// TODO: Add other service types when available
export interface AvailableServicesResponse {
  mongodb?: VersionedService[];
  mysql?: VersionedService[];
  postgresql?: VersionedService[];
}
