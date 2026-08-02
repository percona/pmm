import { type MRT_Row } from 'material-react-table';
import { QueryData, RawQueryData } from 'types/rta.types';
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
