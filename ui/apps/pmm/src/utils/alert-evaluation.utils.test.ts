import { describe, it, expect } from 'vitest';
import {
  AlertEvalFrame,
  GrafanaAlertRuleDefinition,
} from 'types/alerting.types';
import {
  computeValueThreshold,
  parseComparison,
  pickSeriesValue,
  resolveEvalPlan,
} from './alert-evaluation.utils';

describe('parseComparison', () => {
  it('splits a trailing comparison', () => {
    expect(parseComparison('sum(x) / y * 100 > 80')).toEqual({
      lhs: 'sum(x) / y * 100',
      operator: '>',
      threshold: 80,
      isBool: false,
    });
  });

  it('handles each operator', () => {
    expect(parseComparison('a >= 5')?.operator).toBe('>=');
    expect(parseComparison('a < 5')?.operator).toBe('<');
    expect(parseComparison('a <= 5')?.operator).toBe('<=');
  });

  it('flags the bool modifier', () => {
    const parsed = parseComparison('pmm_managed_inventory_agents{} == bool 0 ');
    expect(parsed).toMatchObject({
      operator: '==',
      threshold: 0,
      isBool: true,
    });
  });

  it('returns null without a trailing comparison', () => {
    expect(parseComparison('sum(rate(x[5m]))')).toBeNull();
    expect(parseComparison(undefined)).toBeNull();
  });
});

// Multi-expression rule: A = query, B = constant threshold, C = math `$A > $B`.
const MATH_RULE: GrafanaAlertRuleDefinition = {
  uid: 'efqltck93gn40f',
  condition: 'C',
  data: [
    {
      refId: 'A',
      datasourceUid: 'PA58DA793C7250F1B',
      model: { refId: 'A', expr: 'metric * 100', instant: true },
    },
    {
      refId: 'B',
      datasourceUid: 'PA58DA793C7250F1B',
      model: { refId: 'B', expr: '0.25', instant: true },
    },
    {
      refId: 'C',
      datasourceUid: '__expr__',
      model: { refId: 'C', type: 'math', expression: '$A > $B' },
    },
  ],
};

// Single datasource query with the comparison baked into PromQL.
const SINGLE_QUERY_RULE: GrafanaAlertRuleDefinition = {
  uid: 'bfqkr76psnvnkf',
  condition: 'A',
  data: [
    {
      refId: 'A',
      datasourceUid: 'PA58DA793C7250F1B',
      model: { refId: 'A', expr: 'sum(pg_stat_activity_count) * 100 > 80' },
    },
  ],
};

// Boolean up/down check (equality) — no value/threshold.
const BOOL_RULE: GrafanaAlertRuleDefinition = {
  uid: 'afqkr5oldd5vka',
  condition: 'A',
  data: [
    {
      refId: 'A',
      datasourceUid: 'PA58DA793C7250F1B',
      model: {
        refId: 'A',
        expr: 'pmm_managed_inventory_agents{agent_type="pmm-agent"} == bool 1 ',
      },
    },
  ],
};

// Percentage rule with the `bool` modifier on an inequality — IS gauge-able.
const BOOL_PERCENT_RULE: GrafanaAlertRuleDefinition = {
  uid: 'cfqkr76psnvnkg',
  condition: 'A',
  data: [
    {
      refId: 'A',
      datasourceUid: 'PA58DA793C7250F1B',
      model: {
        refId: 'A',
        expr: 'max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job) mysql_global_variables_max_connections * 100 > bool 90',
      },
    },
  ],
};

// Logical composite (`... and ...`) — multiple comparisons, not a single value/threshold gauge.
const COMPOSITE_RULE: GrafanaAlertRuleDefinition = {
  uid: 'dfqkr76psnvnkh',
  condition: 'A',
  data: [
    {
      refId: 'A',
      datasourceUid: 'PA58DA793C7250F1B',
      model: {
        refId: 'A',
        expr: 'pg_bloat_table_bloat_pct > 70 and on (relname) (pg_bloat_table_real_size > 200000000)',
      },
    },
  ],
};

const THRESHOLD_EXPR_RULE: GrafanaAlertRuleDefinition = {
  uid: 'threshold-rule',
  condition: 'C',
  data: [
    {
      refId: 'A',
      datasourceUid: 'PA58DA793C7250F1B',
      model: { refId: 'A', expr: 'metric' },
    },
    {
      refId: 'C',
      datasourceUid: '__expr__',
      model: {
        refId: 'C',
        type: 'threshold',
        expression: 'A',
        conditions: [{ evaluator: { type: 'lt', params: [10] } }],
      },
    },
  ],
};

describe('resolveEvalPlan', () => {
  it('resolves a math ($A > $B) rule', () => {
    const plan = resolveEvalPlan(MATH_RULE);
    expect(plan).toMatchObject({
      valueRefId: 'A',
      thresholdRefId: 'B',
      operator: '>',
    });
    expect(plan?.data).toHaveLength(3);
  });

  it('resolves a single-query rule by stripping the comparison', () => {
    const plan = resolveEvalPlan(SINGLE_QUERY_RULE);
    expect(plan).toMatchObject({
      valueRefId: 'A',
      operator: '>',
      thresholdConst: 80,
    });
    expect(plan?.data).toHaveLength(1);
    expect(plan?.data[0].model.expr).toBe('sum(pg_stat_activity_count) * 100');
    expect(plan?.data[0].model.instant).toBe(true);
  });

  it('returns null for an equality (up/down) rule', () => {
    expect(resolveEvalPlan(BOOL_RULE)).toBeNull();
  });

  it('resolves a bool-modified inequality (percentage) rule', () => {
    const plan = resolveEvalPlan(BOOL_PERCENT_RULE);
    expect(plan).toMatchObject({
      valueRefId: 'A',
      operator: '>',
      thresholdConst: 90,
    });
    expect(plan?.data[0].model.expr).toBe(
      'max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job) mysql_global_variables_max_connections * 100'
    );
    expect(plan?.data[0].model.instant).toBe(true);
  });

  it('returns null for a logical composite (and/or) rule', () => {
    expect(resolveEvalPlan(COMPOSITE_RULE)).toBeNull();
  });

  it('resolves a threshold expression rule', () => {
    expect(resolveEvalPlan(THRESHOLD_EXPR_RULE)).toMatchObject({
      valueRefId: 'A',
      operator: '<',
      thresholdConst: 10,
    });
  });

  it('returns null when the condition refId is missing', () => {
    expect(
      resolveEvalPlan({ uid: 'x', condition: 'Z', data: MATH_RULE.data })
    ).toBeNull();
  });
});

const frame = (
  value: number,
  labels: Record<string, string> = {}
): AlertEvalFrame => ({
  schema: { fields: [{ labels }] },
  data: { values: [[value]] },
});

describe('pickSeriesValue', () => {
  it('returns the sole frame value', () => {
    expect(pickSeriesValue([frame(0.25)], {})).toBe(0.25);
  });

  it('matches the frame by label subset', () => {
    const frames = [
      frame(10, { service_id: 'a', instance: 'a' }),
      frame(42, { service_id: 'b', instance: 'b' }),
    ];
    expect(
      pickSeriesValue(frames, {
        service_id: 'b',
        instance: 'b',
        severity: 'warning',
      })
    ).toBe(42);
  });

  it('returns null when no frame matches', () => {
    const frames = [
      frame(10, { service_id: 'a' }),
      frame(42, { service_id: 'b' }),
    ];
    expect(pickSeriesValue(frames, { service_id: 'c' })).toBeNull();
    expect(pickSeriesValue([], {})).toBeNull();
    expect(pickSeriesValue(undefined, {})).toBeNull();
  });
});

describe('computeValueThreshold', () => {
  it('computes over direction, floored percent and breaching', () => {
    expect(computeValueThreshold(247, 200, '>')).toEqual({
      value: 247,
      threshold: 200,
      operator: '>',
      direction: 'over',
      breaching: true,
      percent: 23,
    });
  });

  it('computes under direction', () => {
    expect(computeValueThreshold(3, 5, '<')).toMatchObject({
      direction: 'under',
      percent: 40,
    });
  });

  it('derives direction positionally, not from the operator', () => {
    // A `>` rule whose value has dropped below the threshold reads as "under".
    expect(computeValueThreshold(5, 80, '>')).toMatchObject({
      direction: 'under',
      percent: 93,
    });
  });

  it('derives breaching from the operator', () => {
    // `<` rule breaches when the value is below the threshold.
    expect(computeValueThreshold(3, 5, '<').breaching).toBe(true);
    expect(computeValueThreshold(7, 5, '<').breaching).toBe(false);
    // `>` rule whose value dropped below the threshold is no longer breaching.
    expect(computeValueThreshold(5, 80, '>').breaching).toBe(false);
    // `>=`/`<=` boundaries.
    expect(computeValueThreshold(5, 5, '>=').breaching).toBe(true);
    expect(computeValueThreshold(5, 5, '<=').breaching).toBe(true);
  });

  it('returns null percent for a zero threshold', () => {
    expect(computeValueThreshold(5, 0, '>')).toMatchObject({
      direction: 'over',
      percent: null,
    });
  });
});
