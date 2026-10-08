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
  // The agent's explanation of the status: why the session failed to start, or
  // what it cannot collect while running. Absent when there is nothing to say,
  // and always absent from agents that predate it.
  statusMessage?: string;
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
  // Whether the statement is waiting for a lock. UNSPECIFIED means the agent could
  // not read the lock graph at all, which must not be shown as "not blocked";
  // UNATTRIBUTED means the wait found belonged to a later statement on the connection.
  blockedStatus?: BlockedStatus;
  blockedBy?: BlockingTransaction[];
  // The lock the statement itself is waiting for. A property of the waiter: every
  // blocker of a statement is contending over the same requested lock.
  lockedTable?: string;
  // Row locks only. A metadata lock is taken on the table as a whole, so this is
  // empty whenever lockType is metadata.
  lockedIndex?: string;
  lockType?: LockType;
  requestedLockMode?: string;
  // The statement text is incomplete: MySQL keeps only the beginning of a long
  // statement and marks no cut of its own.
  queryTextTruncated?: boolean;
  // Time spent waiting for table locks, as a protobuf duration ("0.000003s").
  // Absent when the server did not measure it.
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
  // The connection was waiting for a lock, but for a statement later than the one sampled:
  // it moved on between the statement read and the lock read. Unlike unspecified, the lock
  // sources were readable, so a refresh rather than a configuration change resolves it.
  unattributed = 'BLOCKED_STATUS_UNATTRIBUTED',
}

// Which of MySQL's two independent locking mechanisms a statement is waiting on. They
// are freed differently — a row lock by ending the holding transaction, a metadata lock
// by the holder finishing its statement or transaction on that table — so the reader has
// to be told which one they are looking at.
export enum LockType {
  unspecified = 'LOCK_TYPE_UNSPECIFIED',
  row = 'LOCK_TYPE_ROW',
  metadata = 'LOCK_TYPE_METADATA',
}

// A transaction preventing a statement from taking the lock it is waiting for.
export interface BlockingTransaction {
  blockingConnId: number | string;
  blockingQuery: string;
  // "Sleep" means the blocker is idle inside an open transaction and is running
  // nothing at all, so blockingQuery is the last statement it ran. The lock may have been
  // taken by an earlier statement in the same transaction.
  blockingCommand: string;
  blockingUsername: string;
  waitDuration?: string | null;
  blockerTransactionDuration?: string | null;
  // The blocker is not itself waiting, so it heads the chain: resolving it
  // releases everything queued behind it.
  root?: boolean;
  // The mode this transaction holds on the contended object: "X,REC_NOT_GAP" and the
  // like for a row lock, "SHARED_READ" and the like for a metadata lock.
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
