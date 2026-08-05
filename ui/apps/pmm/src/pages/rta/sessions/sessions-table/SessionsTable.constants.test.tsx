import { describe, it, expect } from 'vitest';
import { columnId, getSessionsTableColumns } from './SessionsTable.constants';

describe('getSessionsTableColumns', () => {
  it('leaves the table as it was on a single-technology install', () => {
    expect(getSessionsTableColumns(false).map(columnId)).toEqual([
      'sessionName',
      'status',
    ]);
  });

  it('adds Technology after the session name once technologies are mixed', () => {
    expect(getSessionsTableColumns(true).map(columnId)).toEqual([
      'sessionName',
      'serviceType',
      'status',
    ]);
  });

  it('returns a stable reference, so a polled re-render does not rebuild columns', () => {
    expect(getSessionsTableColumns(false)).toBe(getSessionsTableColumns(false));
    expect(getSessionsTableColumns(true)).toBe(getSessionsTableColumns(true));
  });
});
