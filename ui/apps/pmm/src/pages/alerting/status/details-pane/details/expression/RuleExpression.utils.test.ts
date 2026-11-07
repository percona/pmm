import type { GrafanaAlertRuleDefinition } from 'types/alerting.types';
import { getRuleSteps, RuleStepKind } from './RuleExpression.utils';

// The shape pmm-managed stores for a ClickHouse template with an overridable threshold.
const clickHouseRule: GrafanaAlertRuleDefinition = {
  uid: 'rule',
  condition: 'C',
  data: [
    {
      refId: 'A',
      datasourceUid: 'clickhouse-uid',
      queryType: 'timeseries',
      model: { refId: 'A', rawSql: 'SELECT 1', queryType: 'timeseries' },
    },
    {
      refId: 'T_threshold',
      datasourceUid: 'metrics-uid',
      model: {
        refId: 'T_threshold',
        expr: 'max by (node_name) (pmm_alert_threshold_override{rule_id="r"}) or (group by (node_name) (up) * 2)',
      },
    },
    {
      refId: 'B',
      datasourceUid: '__expr__',
      model: { refId: 'B', type: 'reduce', expression: 'A', reducer: 'max' },
    },
    {
      refId: 'C',
      datasourceUid: '__expr__',
      model: { refId: 'C', type: 'math', expression: '$B > $T_threshold' },
    },
  ],
};

describe('getRuleSteps', () => {
  it('lists every step of a ClickHouse rule, collapsing the injected threshold', () => {
    expect(getRuleSteps(clickHouseRule)).toEqual([
      {
        refId: 'A',
        kind: RuleStepKind.SqlQuery,
        content: 'SELECT 1',
        language: 'sql',
        isCondition: false,
      },
      {
        refId: 'T_threshold',
        kind: RuleStepKind.Threshold,
        content: expect.stringContaining('Per-target threshold'),
        language: 'text',
        isCondition: false,
      },
      {
        refId: 'B',
        kind: RuleStepKind.Reduce,
        content: 'max($A)',
        language: 'text',
        isCondition: false,
      },
      {
        refId: 'C',
        kind: RuleStepKind.Math,
        content: '$B > $T_threshold',
        language: 'text',
        isCondition: true,
      },
    ]);
  });

  it('keeps a MetricsQL query and a reduce stored with a `$` ref', () => {
    const steps = getRuleSteps({
      uid: 'rule',
      condition: 'B',
      data: [
        {
          refId: 'A',
          datasourceUid: 'metrics-uid',
          model: { refId: 'A', expr: 'up == 0' },
        },
        {
          refId: 'B',
          datasourceUid: '__expr__',
          model: {
            refId: 'B',
            type: 'reduce',
            expression: '$A',
            reducer: 'last',
          },
        },
      ],
    });

    expect(steps[0]).toMatchObject({
      kind: RuleStepKind.Query,
      content: 'up == 0',
      language: 'promql',
    });
    expect(steps[1]).toMatchObject({
      kind: RuleStepKind.Reduce,
      content: 'last($A)',
      isCondition: true,
    });
  });

  it('falls back to the type of an expression it has no text for', () => {
    const [step] = getRuleSteps({
      uid: 'rule',
      condition: 'C',
      data: [
        {
          refId: 'C',
          datasourceUid: '__expr__',
          model: { refId: 'C', type: 'classic_conditions' },
        },
      ],
    });

    expect(step).toMatchObject({
      kind: RuleStepKind.Expression,
      content: 'classic_conditions',
    });
  });
});
