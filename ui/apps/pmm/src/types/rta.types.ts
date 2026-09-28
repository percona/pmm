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
  // Whether the statement is waiting for a lock. UNSPECIFIED means the agent could
  // not read the lock graph at all, which must not be shown as "not blocked".
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
}

export enum BlockedStatus {
  unspecified = 'BLOCKED_STATUS_UNSPECIFIED',
  notBlocked = 'BLOCKED_STATUS_NOT_BLOCKED',
  blocked = 'BLOCKED_STATUS_BLOCKED',
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
  // nothing at all, so blockingQuery is the statement that took the lock.
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
}

// TODO: Add other service types when available
export interface AvailableServicesResponse {
  mongodb?: VersionedService[];
  mysql?: VersionedService[];
}
