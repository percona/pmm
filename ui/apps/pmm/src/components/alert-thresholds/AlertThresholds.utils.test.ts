import type {
  ListThresholdsResponse,
  PrometheusAlertRulesResponse,
} from 'types/alerting.types';
import type { AlertThresholdRow } from './AlertThresholds.types';
import {
  buildThresholdUpdates,
  getRows,
  getRuleTitles,
} from './AlertThresholds.utils';

const NODE = 'THRESHOLD_SCOPE_NODE' as const;

const rulesResponse = (
  rules: { name: string; labels?: Record<string, string> }[]
): PrometheusAlertRulesResponse =>
  ({
    data: { groups: [{ rules }] },
  }) as PrometheusAlertRulesResponse;

const row = (over: Partial<AlertThresholdRow> = {}): AlertThresholdRow => ({
  id: 'rule-1:threshold:0',
  ruleId: 'rule-1',
  ruleTitle: 'CPU load',
  paramName: 'threshold',
  defaultValue: 80,
  effectiveValue: 80,
  isOverridden: false,
  ...over,
});

describe('getRuleTitles', () => {
  it('indexes rule names by the identity label PMM stamps on them', () => {
    const titles = getRuleTitles(
      rulesResponse([
        { name: 'CPU load', labels: { pmm_rule_id: 'rule-1' } },
        { name: 'Connections', labels: { pmm_rule_id: 'rule-2' } },
      ])
    );

    expect(titles.get('rule-1')).toBe('CPU load');
    expect(titles.get('rule-2')).toBe('Connections');
  });

  it('ignores rules that PMM did not create', () => {
    const titles = getRuleTitles(
      rulesResponse([{ name: 'Someone else rule' }])
    );

    expect(titles.size).toBe(0);
  });
});

describe('getRows', () => {
  // proto3 JSON omits zero values, so a threshold of 0 and a row that is not
  // overridden both arrive with the field absent rather than 0/false. Left uncoerced
  // the table would render blanks instead of numbers.
  it('reads omitted numeric fields as zero rather than undefined', () => {
    const data = {
      thresholds: [{ ruleId: 'rule-1', paramName: 'threshold' }],
    } as ListThresholdsResponse;

    const [first] = getRows(data, new Map());

    expect(first.defaultValue).toBe(0);
    expect(first.effectiveValue).toBe(0);
    expect(first.isOverridden).toBe(false);
  });

  it('joins the rule title, and tolerates a rule that no longer exists', () => {
    const data = {
      thresholds: [
        { ruleId: 'rule-1', paramName: 'threshold' },
        { ruleId: 'deleted-rule', paramName: 'threshold' },
      ],
    } as ListThresholdsResponse;

    const rows = getRows(data, new Map([['rule-1', 'CPU load']]));

    expect(rows[0].ruleTitle).toBe('CPU load');
    expect(rows[1].ruleTitle).toBe('');
  });

  // Two rules duplicated in Grafana share a rule id, which the API explicitly
  // permits, so rule and parameter together do not identify a row. Colliding ids
  // would make one form field drive two rows.
  it('gives duplicated rules distinct row ids', () => {
    const data = {
      thresholds: [
        { ruleId: 'rule-1', paramName: 'threshold' },
        { ruleId: 'rule-1', paramName: 'threshold' },
      ],
    } as ListThresholdsResponse;

    const rows = getRows(data, new Map());

    expect(rows[0].id).not.toBe(rows[1].id);
  });

  it('returns nothing when the response carries no thresholds', () => {
    expect(getRows(undefined, new Map())).toEqual([]);
  });
});

describe('buildThresholdUpdates', () => {
  it('sets a changed value', () => {
    const rows = [row()];

    expect(
      buildThresholdUpdates(rows, { [rows[0].id]: 95 }, NODE, 'node-1')
    ).toEqual([
      {
        scope: NODE,
        target: 'node-1',
        ruleId: 'rule-1',
        paramName: 'threshold',
        value: 95,
      },
    ]);
  });

  it('sends nothing when the value is unchanged', () => {
    const rows = [row({ isOverridden: true, effectiveValue: 95 })];

    expect(
      buildThresholdUpdates(rows, { [rows[0].id]: 95 }, NODE, 'node-1')
    ).toEqual([]);
  });

  // Omitting `value` clears the override. Writing the default as an override instead
  // would pin the target to today's default and stop it following a later change to
  // the rule.
  it('clears by omitting the value when the field is emptied', () => {
    const rows = [row({ isOverridden: true, effectiveValue: 95 })];

    const updates = buildThresholdUpdates(
      rows,
      { [rows[0].id]: undefined },
      NODE,
      'node-1'
    );

    expect(updates).toHaveLength(1);
    expect(updates[0]).not.toHaveProperty('value');
  });

  it('clears when the default is typed back in', () => {
    const rows = [row({ isOverridden: true, effectiveValue: 95 })];

    const updates = buildThresholdUpdates(
      rows,
      { [rows[0].id]: 80 },
      NODE,
      'node-1'
    );

    expect(updates).toHaveLength(1);
    expect(updates[0]).not.toHaveProperty('value');
  });

  it('sends nothing when a row that was never overridden is left at the default', () => {
    const rows = [row()];

    expect(
      buildThresholdUpdates(rows, { [rows[0].id]: 80 }, NODE, 'node-1')
    ).toEqual([]);
  });

  it('treats an emptied string field as a clear, not as zero', () => {
    const rows = [row({ isOverridden: true, effectiveValue: 95 })];

    const updates = buildThresholdUpdates(
      rows,
      { [rows[0].id]: '' as unknown as number },
      NODE,
      'node-1'
    );

    expect(updates).toHaveLength(1);
    expect(updates[0]).not.toHaveProperty('value');
  });

  it('batches a set and a clear from one submission', () => {
    const rows = [
      row({ id: 'a', ruleId: 'rule-1' }),
      row({
        id: 'b',
        ruleId: 'rule-2',
        isOverridden: true,
        effectiveValue: 95,
      }),
    ];

    const updates = buildThresholdUpdates(
      rows,
      { a: 60, b: undefined },
      NODE,
      'node-1'
    );

    expect(updates).toHaveLength(2);
    expect(updates[0]).toHaveProperty('value', 60);
    expect(updates[1]).not.toHaveProperty('value');
  });
});
