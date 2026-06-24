import { describe, expect, it } from 'vitest';
import { DEFAULT_TABLE_STATE } from './tableState.types';
import { mergePerconaTableState } from './tableState.utils';

describe('tableState.utils', () => {
  it('lets urlState override overlapping additionalState keys', () => {
    const merged = mergePerconaTableState(
      {
        ...DEFAULT_TABLE_STATE,
        showColumnFilters: true,
        showGlobalFilter: false,
        pagination: { pageIndex: 2, pageSize: 10 },
      },
      {
        rowSelection: { 'srv-001': true },
        pagination: { pageIndex: 0, pageSize: 25 },
        sorting: [{ id: 'name', desc: true }],
      }
    );

    expect(merged.rowSelection).toEqual({ 'srv-001': true });
    expect(merged.pagination).toEqual({ pageIndex: 2, pageSize: 10 });
    expect(merged.sorting).toEqual([]);
  });
});
