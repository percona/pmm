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
