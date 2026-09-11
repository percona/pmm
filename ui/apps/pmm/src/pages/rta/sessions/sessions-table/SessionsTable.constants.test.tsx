import { describe, it, expect } from 'vitest';
import {
  columnId,
  SESSIONS_TABLE_COLUMNS,
} from './SessionsTable.constants.tsx';

describe('SESSIONS_TABLE_COLUMNS', () => {
  it('names the technology of every session, between the name and the status', () => {
    expect(SESSIONS_TABLE_COLUMNS.map(columnId)).toEqual([
      'sessionName',
      'serviceType',
      'status',
    ]);
  });
});
