import { describe, expect, it } from 'vitest';
import { DEFAULT_TABLE_STATE } from './tableState.types';
import { parseTableUrlState, serializeTableUrlState } from './tableUrlState.utils';

describe('tableUrlState.utils', () => {
  it('parses defaults from an empty query string', () => {
    expect(parseTableUrlState(new URLSearchParams())).toEqual(DEFAULT_TABLE_STATE);
  });

  it('round-trips filters, sort, global filter, and pagination', () => {
    const initial = new URLSearchParams(
      'tab=details&q=mysql&sort=name:desc&f.group=edge&page=2&pageSize=25'
    );
    const state = parseTableUrlState(initial);

    expect(state).toEqual({
      columnFilters: [{ id: 'group', value: 'edge' }],
      globalFilter: 'mysql',
      sorting: [{ id: 'name', desc: true }],
      pagination: { pageIndex: 1, pageSize: 25 },
    });

    const serialized = serializeTableUrlState(state, initial);

    expect(serialized.get('tab')).toBe('details');
    expect(serialized.get('q')).toBe('mysql');
    expect(serialized.get('sort')).toBe('name:desc');
    expect(serialized.get('f.group')).toBe('edge');
    expect(serialized.get('page')).toBe('2');
    expect(serialized.get('pageSize')).toBe('25');
  });

  it('uses paramPrefix for namespaced keys', () => {
    const params = new URLSearchParams('servers.q=prod&servers.sort=status:asc&servers.page=3');
    const state = parseTableUrlState(params, { paramPrefix: 'servers' });

    expect(state.globalFilter).toBe('prod');
    expect(state.sorting).toEqual([{ id: 'status', desc: false }]);
    expect(state.pagination).toEqual({ pageIndex: 2, pageSize: 10 });
  });

  it('parses all filter params from the URL', () => {
    const params = new URLSearchParams('f.group=edge&f.secret=yes');
    const state = parseTableUrlState(params);

    expect(state.columnFilters).toEqual([
      { id: 'group', value: 'edge' },
      { id: 'secret', value: 'yes' },
    ]);
  });

  it('falls back for invalid page values', () => {
    const params = new URLSearchParams('page=0&pageSize=-5');
    const state = parseTableUrlState(params);

    expect(state.pagination).toEqual({ pageIndex: 0, pageSize: 10 });
  });

  it('omits default-equivalent values when serializing', () => {
    const serialized = serializeTableUrlState(DEFAULT_TABLE_STATE, new URLSearchParams('keep=1'));

    expect(serialized.toString()).toBe('keep=1');
  });

  it('respects sync flags', () => {
    const params = new URLSearchParams('q=mysql&sort=name:desc&page=2');
    const state = parseTableUrlState(params, {
      sync: { globalFilter: false, sort: false, pagination: false, filters: true },
    });

    expect(state).toEqual({
      ...DEFAULT_TABLE_STATE,
      columnFilters: [],
    });
  });

  it('round-trips range filter array values', () => {
    const rangeFilters = [{ id: 'cpu', value: ['10', '50'] }];
    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, columnFilters: rangeFilters },
      new URLSearchParams()
    );

    expect(serialized.get('f.cpu')).toBe('["10","50"]');

    const parsed = parseTableUrlState(serialized);
    expect(parsed.columnFilters).toEqual(rangeFilters);
  });

  it('round-trips partial range filter values while typing', () => {
    const partialRange = [{ id: 'memory', value: ['5', ''] }];
    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, columnFilters: partialRange },
      new URLSearchParams()
    );

    expect(serialized.get('f.memory')).toBe('["5",""]');

    const parsed = parseTableUrlState(serialized);
    expect(parsed.columnFilters).toEqual(partialRange);
  });

  it('preserves range tuple length when serializing sparse arrays', () => {
    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, columnFilters: [{ id: 'cpu', value: ['10', undefined] }] },
      new URLSearchParams()
    );

    expect(serialized.get('f.cpu')).toBe('["10",""]');
    expect(parseTableUrlState(serialized).columnFilters).toEqual([
      { id: 'cpu', value: ['10', ''] },
    ]);
  });

  it('omits default sorting from the URL and restores it when the param is absent', () => {
    const defaults = {
      ...DEFAULT_TABLE_STATE,
      sorting: [{ id: 'name', desc: true }],
    };
    const serialized = serializeTableUrlState({ ...defaults }, new URLSearchParams('keep=1'), {
      defaults,
    });

    expect(serialized.get('sort')).toBeNull();
    expect(serialized.get('keep')).toBe('1');

    const parsed = parseTableUrlState(serialized, { defaults });
    expect(parsed.sorting).toEqual(defaults.sorting);
  });

  it('parses an explicit empty sort param as no sorting', () => {
    const defaults = {
      ...DEFAULT_TABLE_STATE,
      sorting: [{ id: 'name', desc: true }],
    };
    const params = new URLSearchParams('sort=');

    expect(parseTableUrlState(params, { defaults }).sorting).toEqual([]);
  });

  it('deep-merges partial pagination defaults', () => {
    const defaults = { pagination: { pageIndex: 0, pageSize: 25 } };
    const parsed = parseTableUrlState(new URLSearchParams(), { defaults });

    expect(parsed.pagination).toEqual({ pageIndex: 0, pageSize: 25 });

    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, pagination: { pageIndex: 0, pageSize: 25 } },
      new URLSearchParams('keep=1'),
      { defaults }
    );

    expect(serialized.get('page')).toBeNull();
    expect(serialized.get('pageSize')).toBeNull();
    expect(serialized.get('keep')).toBe('1');
  });

  it('omits empty range filter values when serializing', () => {
    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, columnFilters: [{ id: 'cpu', value: ['', ''] }] },
      new URLSearchParams('keep=1')
    );

    expect(serialized.get('f.cpu')).toBeNull();
    expect(serialized.get('keep')).toBe('1');
  });
});
