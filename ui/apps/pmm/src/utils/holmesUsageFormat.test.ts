import { aggregateInvestigationUsage, aggregateUsageSeriesByDay, fillDailyCostSeries, resolveDailyCostChartRows } from './holmesUsageFormat';

describe('fillDailyCostSeries', () => {
  it('returns newest day first and drops leading zero-cost days', () => {
    const series = aggregateUsageSeriesByDay([
      { bucket: '2026-05-20', total_cost: 1.5 },
      { bucket: '2026-05-22', total_cost: 0.84 },
    ]);

    const filled = fillDailyCostSeries(series, '2026-04-23T00:00:00.000Z', '2026-05-23T12:00:00.000Z');

    expect(filled[0]?.bucket).toBe('2026-05-23');
    expect(filled[1]?.bucket).toBe('2026-05-22');
    expect(filled.some((row) => row.bucket === '2026-04-23')).toBe(false);
    expect(filled[filled.length - 1]?.bucket).toBe('2026-05-20');
  });

  it('keeps only the latest day when the range has no usage', () => {
    const filled = fillDailyCostSeries([], '2026-05-20T00:00:00.000Z', '2026-05-23T12:00:00.000Z');

    expect(filled).toEqual([expect.objectContaining({ bucket: '2026-05-23', totalCost: 0 })]);
  });
});

describe('resolveDailyCostChartRows', () => {
  it('returns only days with cost for the chart', () => {
    const rows = resolveDailyCostChartRows({
      series: [
        { bucket: '2026-05-20', total_cost: 1.5 },
        { bucket: '2026-05-22', total_cost: 0.84 },
      ],
      events: [],
      fromISO: '2026-04-23T00:00:00.000Z',
      toISO: '2026-05-23T12:00:00.000Z',
    });

    expect(rows.every((row) => row.totalCost > 0)).toBe(true);
    expect(rows.map((row) => row.bucket)).toEqual(['2026-05-22', '2026-05-20']);
  });

  it('falls back to events when summary series is empty but totals exist', () => {
    const rows = resolveDailyCostChartRows({
      series: [],
      events: [
        { created_at: '2026-05-23T10:00:00Z', total_cost: 0.42 },
        { created_at: '2026-05-23T11:00:00Z', total_cost: 0.18 },
      ],
      fromISO: '2026-05-20T00:00:00.000Z',
      toISO: '2026-05-23T12:00:00.000Z',
      total_cost: 0.6,
    });

    expect(rows).toEqual([expect.objectContaining({ bucket: '2026-05-23', totalCost: 0.6 })]);
  });
});

describe('aggregateInvestigationUsage', () => {
  it('prefers API events and merges investigation totals', () => {
    const summary = aggregateInvestigationUsage({
      holmes_call_count: 2,
      holmes_total_tokens: 1200,
      holmes_total_cost: 0.42,
      events: [
        {
          id: 1,
          created_at: '2026-05-23T10:00:00Z',
          feature: 'investigation_run',
          model: 'default',
          total_tokens: 800,
          total_cost: 0.3,
        },
        {
          id: 2,
          created_at: '2026-05-23T10:01:00Z',
          feature: 'investigation_format',
          model: 'default',
          total_tokens: 400,
          total_cost: 0.12,
        },
      ],
    });

    expect(summary.hasUsage).toBe(true);
    expect(summary.callCount).toBe(2);
    expect(summary.totalTokens).toBe(1200);
    expect(summary.totalCost).toBe(0.42);
    expect(summary.steps).toHaveLength(2);
  });

  it('falls back to assistant messages when usage events are unavailable', () => {
    const summary = aggregateInvestigationUsage({
      messages: [
        {
          id: 'm1',
          role: 'assistant',
          created_at: '2026-05-23T10:00:00Z',
          holmes_feature: 'investigation_run',
          model: 'opus',
          total_tokens: 500,
          total_cost: 0.25,
        },
      ],
    });

    expect(summary.hasUsage).toBe(true);
    expect(summary.totalTokens).toBe(500);
    expect(summary.totalCost).toBe(0.25);
    expect(summary.steps).toHaveLength(1);
    expect(summary.steps[0]?.feature).toBe('investigation_run');
  });

  it('merges API events and message rows without duplicates', () => {
    const summary = aggregateInvestigationUsage({
      events: [
        {
          id: 1,
          created_at: '2026-05-23T10:00:00Z',
          feature: 'investigation_run',
          model: 'default',
          total_tokens: 54100,
          cached_tokens: 52000,
          total_cost: 0.426,
        },
      ],
      messages: [
        {
          id: 'm1',
          role: 'assistant',
          created_at: '2026-05-23T10:00:00Z',
          holmes_feature: 'investigation_run',
          model: 'default',
          total_tokens: 54100,
          cached_tokens: 52000,
          total_cost: 0.426,
        },
      ],
    });

    expect(summary.steps).toHaveLength(1);
    expect(summary.totalTokens).toBe(54100);
    expect(summary.totalCost).toBe(0.426);
  });
});
