import { type MRT_Row } from 'material-react-table';
import {
  BlockedStatus,
  BlockingTransaction,
  QueryData,
  RawQueryData,
} from 'types/rta.types';
import { CodeLanguage } from 'types/util.types';
import { parseDuration } from 'utils/duration.utils';

export const queryLanguage = (query: RawQueryData): CodeLanguage =>
  query.mySqlPayload ? 'sql' : 'mongodb';

// MongoDB has no Prism grammar; JavaScript is the closest fit.
export const codeBlockLanguage = (language: CodeLanguage): string =>
  language === 'mongodb' ? 'javascript' : language;

// Non-empty so rows with no value (e.g. a NULL database) can be found via the column filter.
export const UNAVAILABLE_VALUE = 'Unavailable';

export const queryDatabaseName = (query: RawQueryData): string =>
  query.mongoDbPayload?.databaseName ||
  query.mySqlPayload?.databaseName ||
  UNAVAILABLE_VALUE;

export const queryUsername = (query: RawQueryData): string =>
  query.mongoDbPayload?.username ||
  query.mySqlPayload?.username ||
  UNAVAILABLE_VALUE;

// Keeps milliseconds below 10s, where most live statements would otherwise read as "0";
// shared with the details pane so both show the same number.
export const elapsedTimeValue = (seconds: number): string => {
  const rounded = Math.round(seconds * 1000) / 1000;

  return rounded < 10 ? rounded.toFixed(3) : String(Math.round(rounded));
};

// SI unit rather than "seconds" to fit the narrow overview column.
export const formatElapsedTime = (seconds: number): string =>
  `${elapsedTimeValue(seconds)} s`;

export const lockTimeMs = (lockTime?: string | null): number | undefined =>
  lockTime ? parseDuration(lockTime) : undefined;

// MySQL measures lock time in microseconds; three decimals keep a 3µs wait visible.
export const formatLockTimeMs = (lockTime?: string | null): string => {
  const ms = lockTimeMs(lockTime);

  return ms === undefined ? '' : `${Number(ms.toFixed(3))} ms`;
};

// Can be hidden: they add little and can dominate the list under transactional workloads.
const TRANSACTION_CONTROL_STATEMENTS = new Set([
  'COMMIT',
  'COMMIT WORK',
  'ROLLBACK',
  'ROLLBACK WORK',
  'BEGIN',
  'BEGIN WORK',
  'START TRANSACTION',
]);

// Matches bare statements only: "COMMIT;" matches, "COMMIT AND CHAIN" stays visible.
export const isTransactionControl = (query: RawQueryData): boolean => {
  const normalized = query.queryText
    .replace(/;+\s*$/, '')
    .trim()
    .replace(/\s+/g, ' ')
    .toUpperCase();
  return TRANSACTION_CONTROL_STATEMENTS.has(normalized);
};

// Comma-separated terms, since a value picker would not scale to thousands of values.
export const filterCommaSeparated = (
  row: MRT_Row<QueryData>,
  id: string,
  filterValue: string
) => {
  const terms = String(filterValue)
    .split(',')
    .map((term) => term.trim().toLowerCase())
    .filter(Boolean);

  if (!terms.length) {
    return true;
  }

  const value = String(row.getValue(id) ?? '').toLowerCase();

  return terms.some((term) => value.includes(term));
};

// What may sit in the field: a non-negative decimal, or a prefix of one. A
// lone '.' is on its way to '.5', so it has to be typeable.
const ELAPSED_TIME_INPUT = /^\d*\.?\d*$/;
// What the filter can actually use: a number of seconds. '' and '.' are not.
const ELAPSED_TIME_BOUND = /^(?:\d+(?:\.\d*)?|\.\d+)$/;

export const isElapsedTimeBoundInput = (value: string) =>
  ELAPSED_TIME_INPUT.test(value);

export const toElapsedTimeBound = (value: string) =>
  ELAPSED_TIME_BOUND.test(value) ? value : '';

export const filterElapsedTime = (
  row: MRT_Row<QueryData>,
  id: string,
  filterValue: [string, string]
) => {
  const [min, max] = filterValue;
  const valueSeconds = row.getValue<number>(id);
  if (valueSeconds === null || valueSeconds === undefined) return false;

  const minSet = min !== '' && min != null && !Number.isNaN(parseFloat(min));
  const maxSet = max !== '' && max != null && !Number.isNaN(parseFloat(max));

  if (!minSet && !maxSet) return true;

  if (minSet && !maxSet) return valueSeconds >= parseFloat(min);
  if (!minSet && maxSet) return valueSeconds <= parseFloat(max);

  return valueSeconds >= parseFloat(min) && valueSeconds <= parseFloat(max);
};

export const isBlocked = (query: RawQueryData): boolean =>
  query.mySqlPayload?.blockedStatus === BlockedStatus.blocked;

// Kept apart from "not blocked" so a monitoring gap does not look like a healthy server.
export const isBlockingUnknown = (query: RawQueryData): boolean =>
  !!query.mySqlPayload &&
  (query.mySqlPayload.blockedStatus === undefined ||
    query.mySqlPayload.blockedStatus === BlockedStatus.unspecified);

// Thread state of a metadata-lock or table-lock wait; row-lock waits have no such state.
const LOCK_WAIT_STATE = /^Waiting for .*lock$/i;

// A lock wait whose holder PMM cannot read (the metadata-lock instrument is off by default on
// MySQL 5.7 and MariaDB); trusts only the thread state, as most undecided rows are not waiting.
export const isLockWaitUnreadable = (query: RawQueryData): boolean =>
  isBlockingUnknown(query) &&
  LOCK_WAIT_STATE.test(query.mySqlPayload?.state?.trim() ?? '');

// Unlike isBlockingUnknown, a refresh resolves this rather than a configuration change.
export const isBlockingUnattributed = (query: RawQueryData): boolean =>
  query.mySqlPayload?.blockedStatus === BlockedStatus.unattributed;

// Blockers that are not themselves waiting; several can hold up one statement at once.
export const blockingRoots = (
  blockers: BlockingTransaction[]
): BlockingTransaction[] => blockers.filter((blocker) => blocker.root);

// Shared by the table chip, CSV column and details pane so they agree on who is to blame.
export const soleBlockerOf = (
  blockers: BlockingTransaction[]
): BlockingTransaction | undefined => {
  const roots = blockingRoots(blockers);

  if (roots.length === 1) {
    return roots[0];
  }

  // No root means a cycle or a partly read lock graph, so no single blocker can be blamed.
  return undefined;
};

export const soleBlocker = (
  query: RawQueryData
): BlockingTransaction | undefined =>
  soleBlockerOf(query.mySqlPayload?.blockedBy ?? []);

// query_id is a MySQL connection id, unique only within one instance.
export const rtaRowId = (query: RawQueryData): string =>
  `${query.serviceId}:${query.queryId}`;

// A connection keeps its id across statements; a running one keeps its text and only grows older.
export const isSameStatement = (
  selected: QueryData,
  candidate: QueryData
): boolean =>
  rtaRowId(selected) === rtaRowId(candidate) &&
  selected.queryText === candidate.queryText &&
  (candidate.queryExecutionDurationMs ?? 0) >=
    (selected.queryExecutionDurationMs ?? 0);

// Anchors the details pane's previous/next on the statement, not on its connection.
export const statementRowId = (query: RawQueryData): string =>
  `${rtaRowId(query)}:${query.queryText}`;
