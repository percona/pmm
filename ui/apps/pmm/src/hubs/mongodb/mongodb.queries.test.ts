import {
  deriveQueries,
  findRunningQueries,
  findSlowerQueries,
  formatCalls,
  formatRatio,
} from './mongodb.queries';
import type { QanRowInput, RtaQueryInput } from './mongodb.types';
import { service } from './mongodb.testUtils';

const row = (
  queryId: string,
  avgQueryTime: number,
  numQueries = 1200
): QanRowInput => ({
  queryId,
  fingerprint: `db.${queryId}.find()`,
  numQueries,
  avgQueryTime,
});

const rta = (serviceId: string, minutes: number): RtaQueryInput => ({
  serviceId,
  serviceName: serviceId,
  queryText: 'db.orders.aggregate([])',
  operation: 'aggregate',
  namespace: 'shop.orders',
  durationMs: minutes * 60_000,
});

describe('findSlowerQueries', () => {
  it('flags a query at least 3× slower than usual', () => {
    const [slower] = findSlowerQueries([row('a', 0.84)], [row('a', 0.14)]);

    expect(slower.ratio).toBeCloseTo(6);
    expect(findSlowerQueries([row('a', 0.29)], [row('a', 0.1)])).toEqual([]);
  });

  it('needs 50 calls in the last hour', () => {
    expect(findSlowerQueries([row('a', 1, 49)], [row('a', 0.1)])).toEqual([]);
    expect(findSlowerQueries([row('a', 1, 50)], [row('a', 0.1)])).toHaveLength(
      1
    );
  });

  it('needs an average of at least 0.1 s', () => {
    expect(findSlowerQueries([row('a', 0.09)], [row('a', 0.01)])).toEqual([]);
  });

  it('ignores a query without a baseline', () => {
    expect(findSlowerQueries([row('new', 5)], [])).toEqual([]);
    expect(findSlowerQueries([row('a', 5)], [row('a', 0)])).toEqual([]);
  });

  it('puts the biggest slowdown first', () => {
    expect(
      findSlowerQueries(
        [row('a', 0.4), row('b', 1)],
        [row('a', 0.1), row('b', 0.1)]
      ).map(({ query }) => query.queryId)
    ).toEqual(['b', 'a']);
  });
});

describe('findRunningQueries', () => {
  const services = [service('s1'), service('s2')];

  it('finds queries running a minute or more, only with an active session', () => {
    const found = findRunningQueries(
      {
        sessionServiceIds: ['svc-s1'],
        queries: [rta('svc-s1', 14), rta('svc-s1', 0.5), rta('svc-s2', 20)],
      },
      services
    );

    expect(found.map(({ query }) => query.durationMs)).toEqual([14 * 60_000]);
  });

  it('skips services outside the shown estate', () => {
    expect(
      findRunningQueries(
        { sessionServiceIds: ['svc-x'], queries: [rta('svc-x', 5)] },
        services
      )
    ).toEqual([]);
  });
});

describe('deriveQueries', () => {
  it('has no rows without query data', () => {
    const { queries } = deriveQueries(
      { qan: null, realtime: null },
      [],
      'prod'
    );

    expect(queries).toEqual({ slower: [] });
  });

  it('builds rows with their facts and doors', () => {
    const { queries, running } = deriveQueries(
      {
        qan: { current: [row('Q1', 0.84)], baseline: [row('Q1', 0.14)] },
        realtime: {
          sessionServiceIds: ['svc-s1'],
          queries: [rta('svc-s1', 14)],
        },
      },
      [service('s1')],
      null
    );

    expect(queries.slower).toEqual([
      {
        id: 'slower:Q1',
        text: 'db.Q1.find()',
        headline: '6× slower than usual',
        detail: '0.84s avg · usual 0.14s · 1.2k calls in 1h',
        severity: 'warning',
        door: {
          label: 'Open in Query analytics',
          to: '/graph/d/pmm-qan/pmm-query-analytics?var-service_type=mongodb&from=now-1h&to=now&filter_by=Q1&query_selected=true&details_tab=details',
        },
      },
    ]);
    // Queries running long go to Needs attention instead.
    expect(queries).not.toHaveProperty('runningLong');
    expect(running.map(({ query }) => query.durationMs)).toEqual([
      14 * 60 * 1000,
    ]);
  });

  it('shows at most five rows per list', () => {
    const current = Array.from({ length: 7 }, (_, index) =>
      row(`q${index}`, 1)
    );
    const baseline = current.map((item) => ({ ...item, avgQueryTime: 0.1 }));

    expect(
      deriveQueries({ qan: { current, baseline }, realtime: null }, [], null)
        .queries.slower
    ).toHaveLength(5);
  });
});

describe('formatters', () => {
  it('writes ratios and call counts compactly', () => {
    expect(formatRatio(5.9999)).toBe('6');
    expect(formatRatio(3.46)).toBe('3.5');
    expect(formatRatio(12.4)).toBe('12');
    expect(formatCalls(1200)).toBe('1.2k');
    expect(formatCalls(42)).toBe('42');
  });
});
