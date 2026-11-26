import type {
  GrafanaAlertQuery,
  GrafanaAlertRuleDefinition,
} from 'types/alerting.types';
import { Messages } from '../AlertDetailsTab.messages';

// Grafana's server-side expressions identify themselves by this datasource UID.
const EXPRESSION_DATASOURCE_UID = '__expr__';

// The metric read by the per-target threshold step pmm-managed injects for overridable params.
const THRESHOLD_OVERRIDE_METRIC = 'pmm_alert_threshold_override';

export enum RuleStepKind {
  Query = 'query',
  SqlQuery = 'sqlQuery',
  Threshold = 'threshold',
  Reduce = 'reduce',
  Math = 'math',
  Expression = 'expression',
}

export interface RuleStep {
  refId: string;
  kind: RuleStepKind;
  content: string;
  language: string;
  isCondition: boolean;
}

const toStep = ({
  datasourceUid,
  model,
}: GrafanaAlertQuery): Omit<RuleStep, 'refId' | 'isCondition'> => {
  if (datasourceUid === EXPRESSION_DATASOURCE_UID) {
    const expression = model.expression ?? '';

    if (model.type === 'reduce') {
      // pmm-managed stores the input as a bare ref, Grafana's editor as `$A`.
      const input = expression.replace(/^\$/, '');
      return {
        kind: RuleStepKind.Reduce,
        content: `${String(model.reducer)}($${input})`,
        language: 'text',
      };
    }

    return {
      kind: model.type === 'math' ? RuleStepKind.Math : RuleStepKind.Expression,
      content: expression || String(model.type ?? ''),
      language: 'text',
    };
  }

  if (typeof model.rawSql === 'string') {
    return {
      kind: RuleStepKind.SqlQuery,
      content: model.rawSql,
      language: 'sql',
    };
  }

  const expr = model.expr ?? '';
  if (expr.includes(THRESHOLD_OVERRIDE_METRIC)) {
    return {
      kind: RuleStepKind.Threshold,
      content: Messages.details.thresholdStep,
      language: 'text',
    };
  }

  return { kind: RuleStepKind.Query, content: expr, language: 'promql' };
};

// Lists a rule's steps for display, in the order Grafana evaluates them.
export const getRuleSteps = (
  definition: GrafanaAlertRuleDefinition
): RuleStep[] =>
  definition.data.map((query) => ({
    ...toStep(query),
    refId: query.refId,
    isCondition: query.refId === definition.condition,
  }));
