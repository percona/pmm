import { type MRT_Row } from 'material-react-table';
import {
  BlockedStatus,
  BlockingTransaction,
  QueryData,
  RawQueryData,
} from 'types/rta.types';
import { CodeLanguage } from 'types/util.types';

// queryLanguage returns the syntax-highlighting language for a query
// based on which database-specific payload it carries.
export const queryLanguage = (query: RawQueryData): CodeLanguage =>
  query.mySqlPayload ? 'sql' : 'mongodb';

// codeBlockLanguage maps a CodeLanguage to a Prism language understood by the
// CodeBlock component. MongoDB has no Prism grammar; JavaScript is the closest fit.
export const codeBlockLanguage = (language: CodeLanguage): string =>
  language === 'mongodb' ? 'javascript' : language;

// Sentinel for rows where the database reports no value (e.g. a MySQL
// connection that never ran USE has a NULL database). Using a non-empty
// label keeps the cell readable and lets such rows be found by typing
// it into the column filter.
export const UNAVAILABLE_VALUE = 'Unavailable';

// Fields common to all database types are resolved from whichever payload is
// present.
export const queryDatabaseName = (query: RawQueryData): string =>
  query.mongoDbPayload?.databaseName ||
  query.mySqlPayload?.databaseName ||
  UNAVAILABLE_VALUE;

export const queryUsername = (query: RawQueryData): string =>
  query.mongoDbPayload?.username ||
  query.mySqlPayload?.username ||
  UNAVAILABLE_VALUE;

// elapsedTimeValue rounds a duration in seconds for display. Below 10s it keeps
// millisecond precision, because most statements caught by a live view finish in
// a few milliseconds and would otherwise all read as "0"; above 10s the
// milliseconds carry no information and only widen the column. Shared with the
// details pane so both render the same number for the same query.
export const elapsedTimeValue = (seconds: number): string => {
  const rounded = Math.round(seconds * 1000) / 1000;

  return rounded < 10 ? rounded.toFixed(3) : String(Math.round(rounded));
};

// formatElapsedTime renders the value with the SI unit instead of the "seconds"
// word, for the width-constrained overview column.
export const formatElapsedTime = (seconds: number): string =>
  `${elapsedTimeValue(seconds)}s`;

// Transaction-control statements that add little value to Real-Time Analytics
// and can dominate the list under transactional workloads (e.g. sysbench).
// "WORK" is the optional SQL keyword (COMMIT WORK / ROLLBACK WORK).
const TRANSACTION_CONTROL_STATEMENTS = new Set([
  'COMMIT',
  'COMMIT WORK',
  'ROLLBACK',
  'ROLLBACK WORK',
  'BEGIN',
  'BEGIN WORK',
  'START TRANSACTION',
]);

// isTransactionControl reports whether a query is a bare transaction-control statement.
// The text is normalized (trailing semicolons removed, internal whitespace collapsed,
// upper-cased) so variations like "COMMIT;" or "START  TRANSACTION" are matched, while
// compound statements such as "COMMIT AND CHAIN" are left visible.
export const isTransactionControl = (query: RawQueryData): boolean => {
  const normalized = query.queryText
    .replace(/;+\s*$/, '')
    .trim()
    .replace(/\s+/g, ' ')
    .toUpperCase();
  return TRANSACTION_CONTROL_STATEMENTS.has(normalized);
};

// filterCommaSeparated matches a cell against a comma-separated list of
// case-insensitive substrings; a row passes when any term matches. This keeps
// the Database/User filters usable with thousands of distinct values, where a
// value picker would not scale.
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

// isBlocked reports whether a statement is known to be waiting for a row lock. Only MySQL
// reports this; MongoDB rows are never blocked as far as RTA is concerned.
export const isBlocked = (query: RawQueryData): boolean =>
  query.mySqlPayload?.blockedStatus === BlockedStatus.blocked;

// isBlockingUnknown reports that the agent could not read the lock graph for this statement,
// so nothing is known about whether it is waiting. Distinct from a verified "not blocked":
// showing those the same way would let a monitoring gap look like a healthy server.
export const isBlockingUnknown = (query: RawQueryData): boolean =>
  !!query.mySqlPayload &&
  (query.mySqlPayload.blockedStatus === undefined ||
    query.mySqlPayload.blockedStatus === BlockedStatus.unspecified);

// blockingRoots returns the blockers that are not themselves waiting. Several can hold up one
// statement at once, so this is a list: naming one of them as the culprit would be wrong
// whenever there is more than one.
export const blockingRoots = (
  blockers: BlockingTransaction[]
): BlockingTransaction[] => blockers.filter((blocker) => blocker.root);

// soleBlockerOf returns the one transaction responsible for a wait, and nothing when the
// answer is not a single transaction. Shared by every surface that has room for one name --
// the table chip, the CSV column, the details pane -- so they cannot disagree about who is
// to blame for the same row.
export const soleBlockerOf = (
  blockers: BlockingTransaction[]
): BlockingTransaction | undefined => {
  const roots = blockingRoots(blockers);

  // Exactly one transaction is holding the statement up and is not itself waiting.
  if (roots.length === 1) {
    return roots[0];
  }

  // No root at all means every participant is waiting -- a cycle, or a graph we only read
  // part of. Naming the single survivor would claim resolving it frees the statement, which
  // is only true if it really is the whole story, and we cannot tell those apart. So do not.
  return undefined;
};

export const soleBlocker = (
  query: RawQueryData
): BlockingTransaction | undefined =>
  soleBlockerOf(query.mySqlPayload?.blockedBy ?? []);

// rtaRowId identifies a row across every service being watched at once.
//
// query_id is the database's own identifier -- a MySQL connection id -- and is unique only
// within one instance. MySQL hands them out from 1 and starts over when the server restarts,
// so two instances of similar age share almost all of their ids. The overview watches several
// services at a time, and the details pane finds the selected row by matching this id, so
// keying on it alone makes the pane navigate to a row on the wrong server.
export const rtaRowId = (query: RawQueryData): string =>
  `${query.serviceId}:${query.queryId}`;
