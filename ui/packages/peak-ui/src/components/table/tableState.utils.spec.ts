import { MRT_ColumnFiltersState } from 'material-react-table';
import { describe, expect, it } from 'vitest';
import { DEFAULT_TABLE_STATE } from './tableState.types';
import {
  cloneColumnFilters,
  mergePerconaTableState,
  stableDependencyKey,
} from './tableState.utils';

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

  it('stableDependencyKey does not throw for circular additionalState values', () => {
    const circular: Record<string, unknown> = { rowSelection: { 'srv-001': true } };
    circular.self = circular;

    expect(() => stableDependencyKey(circular)).not.toThrow();
    expect(stableDependencyKey(circular)).toContain('rowSelection');
  });

  it('stableDependencyKey is invariant to object key order', () => {
    expect(stableDependencyKey({ a: 1, b: 2 })).toBe(stableDependencyKey({ b: 2, a: 1 }));
  });

  it('cloneColumnFilters clones range filter value tuples', () => {
    const filters: MRT_ColumnFiltersState = [{ id: 'cpu', value: ['10', ''] }];
    const cloned = cloneColumnFilters(filters);

    expect(cloned).toEqual(filters);
    expect(cloned[0].value).not.toBe(filters[0].value);
  });
});
