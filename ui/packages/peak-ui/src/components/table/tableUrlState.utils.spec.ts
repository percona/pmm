import { describe, expect, it } from 'vitest';
import { DEFAULT_TABLE_STATE } from './tableState.types';
import {
  normalizeSearchParamsKey,
  parseTableUrlState,
  serializeTableUrlState,
} from './tableUrlState.utils';

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

  it('preserves multi-value array filter length when serializing', () => {
    const multiSelectFilters = [{ id: 'tags', value: ['edge', 'prod'] }];
    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, columnFilters: multiSelectFilters },
      new URLSearchParams()
    );

    expect(serialized.get('f.tags')).toBe('["edge","prod"]');
    expect(parseTableUrlState(serialized).columnFilters).toEqual(multiSelectFilters);
  });

  it('preserves single-value array filter length when serializing', () => {
    const singleValueArray = [{ id: 'group', value: ['edge'] }];
    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, columnFilters: singleValueArray },
      new URLSearchParams()
    );

    expect(serialized.get('f.group')).toBe('["edge"]');
    expect(parseTableUrlState(serialized).columnFilters).toEqual(singleValueArray);
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

  it('falls back to defaults.columnFilters when the URL has no filter params', () => {
    const defaults = {
      ...DEFAULT_TABLE_STATE,
      columnFilters: [{ id: 'group', value: 'edge' }],
    };

    expect(parseTableUrlState(new URLSearchParams('q=mysql'), { defaults }).columnFilters).toEqual(
      defaults.columnFilters
    );
    expect(
      parseTableUrlState(new URLSearchParams('f.group=prod'), { defaults }).columnFilters
    ).toEqual([{ id: 'group', value: 'prod' }]);
  });

  it('preserves unsynced query params when serializing synced slices', () => {
    const initial = new URLSearchParams(
      'q=mysql&sort=name:desc&page=2&pageSize=25&f.group=edge&keep=1'
    );
    const state = {
      ...DEFAULT_TABLE_STATE,
      columnFilters: [{ id: 'group', value: 'prod' }],
    };

    const serialized = serializeTableUrlState(state, initial, {
      sync: { globalFilter: false, sort: false, pagination: false, filters: true },
    });

    expect(serialized.get('q')).toBe('mysql');
    expect(serialized.get('sort')).toBe('name:desc');
    expect(serialized.get('page')).toBe('2');
    expect(serialized.get('pageSize')).toBe('25');
    expect(serialized.get('f.group')).toBe('prod');
    expect(serialized.get('keep')).toBe('1');
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

  it('writes an empty sort param when clearing a non-empty default sort', () => {
    const defaults = {
      ...DEFAULT_TABLE_STATE,
      sorting: [{ id: 'name', desc: true }],
    };
    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, sorting: [] },
      new URLSearchParams('keep=1'),
      { defaults }
    );

    expect(serialized.get('sort')).toBe('');
    expect(parseTableUrlState(serialized, { defaults }).sorting).toEqual([]);
  });

  it('writes an empty globalFilter param when clearing a non-empty default', () => {
    const defaults = {
      ...DEFAULT_TABLE_STATE,
      globalFilter: 'mysql',
    };
    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, globalFilter: '' },
      new URLSearchParams('keep=1'),
      { defaults }
    );

    expect(serialized.get('q')).toBe('');
    expect(parseTableUrlState(serialized, { defaults }).globalFilter).toBe('');
  });

  it('writes an explicit empty filter marker when clearing non-empty default filters', () => {
    const defaults = {
      ...DEFAULT_TABLE_STATE,
      columnFilters: [{ id: 'group', value: 'edge' }],
    };
    const serialized = serializeTableUrlState(
      { ...DEFAULT_TABLE_STATE, columnFilters: [] },
      new URLSearchParams('keep=1'),
      { defaults }
    );

    expect(serialized.get('f._')).toBe('1');
    expect(parseTableUrlState(serialized, { defaults }).columnFilters).toEqual([]);
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

describe('normalizeSearchParamsKey', () => {
  it('preserves ampersands inside param values', () => {
    const withAmpersand = new URLSearchParams([['q', 'a&b']]);
    const withoutAmpersand = new URLSearchParams([['q', 'ab']]);

    expect(normalizeSearchParamsKey(withAmpersand)).not.toBe(
      normalizeSearchParamsKey(withoutAmpersand)
    );
    expect(normalizeSearchParamsKey(withAmpersand)).toContain('a&b');
  });

  it('is order-independent', () => {
    const first = new URLSearchParams('b=2&a=1');
    const second = new URLSearchParams('a=1&b=2');

    expect(normalizeSearchParamsKey(first)).toBe(normalizeSearchParamsKey(second));
  });
});
