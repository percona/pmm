import {
  filterFromSearchParams,
  hasFilters,
  rangeFromSearchParams,
  searchParamsFromFilter,
  severityClassOf,
  toggleSeverityClass,
} from './logsFilter';

const now = new Date('2026-10-17T12:00:00Z');

describe('logsFilter', () => {
  it('round-trips every key', () => {
    const params = new URLSearchParams(
      'from=1760695200000&to=1760698800000&node=n1&node=n2&exclude_node=pmm-server&source=pmm-managed&preset=postgres&preset=mysql_error&severity=13&severity_max=16&q=deadlock&pattern=connection+%3F&hide=7&hide=18446744073709551615'
    );
    const filter = filterFromSearchParams(params, now);
    expect(filter).toEqual({
      start: '2025-10-17T10:00:00.000Z',
      end: '2025-10-17T11:00:00.000Z',
      nodeNames: ['n1', 'n2'],
      excludeNodeNames: ['pmm-server'],
      sources: ['pmm-managed'],
      presets: ['postgres', 'mysql_error'],
      minSeverity: 13,
      maxSeverity: 16,
      text: 'deadlock',
      pattern: 'connection ?',
      excludePatternHashes: ['7', '18446744073709551615'],
    });
    expect(rangeFromSearchParams(params)).toBeUndefined();
    expect(searchParamsFromFilter(filter).toString()).toBe(params.toString());
  });

  it('turns a quick range into a window ending now', () => {
    const params = new URLSearchParams('range=24h');
    expect(filterFromSearchParams(params, now)).toEqual({
      start: '2026-10-16T12:00:00.000Z',
      end: '2026-10-17T12:00:00.000Z',
    });
    expect(rangeFromSearchParams(params)).toBe('24h');
    const filter = filterFromSearchParams(params, now);
    expect(searchParamsFromFilter(filter, '24h').toString()).toBe('range=24h');
  });

  it('defaults to the last hour', () => {
    const params = new URLSearchParams('range=bogus');
    expect(filterFromSearchParams(params, now).start).toBe(
      '2026-10-17T11:00:00.000Z'
    );
    expect(rangeFromSearchParams(new URLSearchParams())).toBe('1h');
  });

  it('prefers absolute from/to over range', () => {
    const params = new URLSearchParams(
      'range=7d&from=1760695200000&to=1760698800000'
    );
    expect(filterFromSearchParams(params, now).start).toBe(
      '2025-10-17T10:00:00.000Z'
    );
    expect(rangeFromSearchParams(params)).toBeUndefined();
  });

  it('selects one severity class and clears it on a second click', () => {
    const base = { start: 's', end: 'e', text: 'x' };
    const warnings = toggleSeverityClass(base, 'warning');
    expect(warnings).toEqual({ ...base, minSeverity: 13, maxSeverity: 16 });
    expect(severityClassOf(warnings)).toBe('warning');
    expect(toggleSeverityClass(warnings, 'warning')).toEqual(base);
    expect(toggleSeverityClass(base, 'other')).toEqual({
      ...base,
      maxSeverity: 8,
    });
    expect(toggleSeverityClass(warnings, 'error')).toEqual({
      ...base,
      minSeverity: 17,
    });
    expect(severityClassOf({ ...base, minSeverity: 13 })).toBeUndefined();
  });

  it('counts hidden patterns as a filter', () => {
    expect(hasFilters({ start: 's', end: 'e' })).toBe(false);
    expect(
      hasFilters({ start: 's', end: 'e', excludePatternHashes: ['7'] })
    ).toBe(true);
  });

  it('ignores an unknown severity', () => {
    const params = new URLSearchParams('severity=12');
    expect(filterFromSearchParams(params, now).minSeverity).toBeUndefined();
  });
});
