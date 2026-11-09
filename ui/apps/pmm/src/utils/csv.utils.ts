import { format } from 'date-fns';
import { download, generateCsv, mkConfig } from 'export-to-csv';

const CSV_FORMULA_PREFIX = /^[=+\-@\t\r]/;

export type CsvValue = string | number | boolean;
export type CsvRow = Record<string, CsvValue>;

export const sanitizeCsvCell = (value: string): string => {
  if (CSV_FORMULA_PREFIX.test(value)) {
    return `'${value}`;
  }

  return value;
};

// export-to-csv rejects object values outright, so anything that is not a
// primitive has to be flattened or serialized before it reaches the generator.
export const toCsvValue = (value: unknown): CsvValue => {
  if (value === null || value === undefined) {
    return '';
  }

  if (typeof value === 'string') {
    return sanitizeCsvCell(value);
  }

  if (typeof value === 'number' || typeof value === 'boolean') {
    return value;
  }

  return sanitizeCsvCell(JSON.stringify(value) ?? '');
};

// Columns are collected across every row: export-to-csv derives headers from
// the first row only, which would drop fields that are unset on it. Listed
// columns come first, in order; the rest follow in the order they appear.
export const collectCsvColumns = (
  rows: CsvRow[],
  order: string[] = []
): string[] => {
  const rank = (column: string) => {
    const index = order.indexOf(column);

    return index === -1 ? order.length : index;
  };
  const columns = [...new Set(rows.flatMap((row) => Object.keys(row)))];

  // Sorting is stable, so unlisted columns share a rank and keep their order
  return columns.sort((a, b) => rank(a) - rank(b));
};

export const buildCsvFilename = (prefix: string, date = new Date()): string =>
  `${prefix}_${format(date, 'yyyyMMdd_HHmmss')}`;

export const downloadCsv = (
  rows: CsvRow[],
  filenamePrefix: string,
  columnOrder?: string[]
): void => {
  if (rows.length === 0) {
    return;
  }

  const csvConfig = mkConfig({
    columnHeaders: collectCsvColumns(rows, columnOrder),
    filename: buildCsvFilename(filenamePrefix),
  });
  const csv = generateCsv(csvConfig)(rows);
  download(csvConfig)(csv);
};
