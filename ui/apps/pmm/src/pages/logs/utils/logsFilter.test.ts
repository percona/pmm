import {
  filterFromSearchParams,
  rangeFromSearchParams,
  searchParamsFromFilter,
} from './logsFilter';

const now = new Date('2026-10-17T12:00:00Z');

describe('logsFilter', () => {
  it('round-trips every key', () => {
    const params = new URLSearchParams(
      'from=1760695200000&to=1760698800000&node=n1&node=n2&source=pmm-managed&preset=postgres&preset=mysql_error&severity=13&q=deadlock&pattern=connection+%3F'
    );
    const filter = filterFromSearchParams(params, now);
    expect(filter).toEqual({
      start: '2025-10-17T10:00:00.000Z',
      end: '2025-10-17T11:00:00.000Z',
      nodeNames: ['n1', 'n2'],
      sources: ['pmm-managed'],
      presets: ['postgres', 'mysql_error'],
      minSeverity: 13,
      text: 'deadlock',
      pattern: 'connection ?',
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

  it('ignores an unknown severity', () => {
    const params = new URLSearchParams('severity=12');
    expect(filterFromSearchParams(params, now).minSeverity).toBeUndefined();
  });
});
