import {
  AlertEvalFrame,
  GrafanaAlertQuery,
  GrafanaAlertRuleDefinition,
} from 'types/alerting.types';

export type ThresholdOperator = '>' | '>=' | '<' | '<=' | '==' | '!=';

export interface ParsedComparison {
  lhs: string;
  operator: ThresholdOperator;
  threshold: number;
  isBool: boolean;
}

export interface EvalPlan {
  // Queries to send to POST /v1/eval.
  data: GrafanaAlertQuery[];
  // refId whose evaluated result is the current value.
  valueRefId: string;
  // refId whose evaluated result is the threshold (multi-expression rules)...
  thresholdRefId?: string;
  // ...or a constant threshold known up-front (single-query / classic rules).
  thresholdConst?: number;
  operator: ThresholdOperator;
}

export interface ValueThresholdResult {
  value: number;
  threshold: number;
  operator: ThresholdOperator;
  direction: 'over' | 'under';
  percent: number | null;
}

const EXPR_DATASOURCE = '__expr__';

// Trailing PromQL comparison: `<lhs> <op> [bool] <number>` at the end of the query.
const TRAILING_COMPARISON =
  /^([\s\S]*?)\s*(>=|<=|==|!=|>|<)\s*(bool\s+)?(-?\d+(?:\.\d+)?)\s*$/;

// `$A > $B` style condition inside a Grafana math expression.
const MATH_COMPARISON =
  /\$\{?(\w+)\}?\s*(>=|<=|==|!=|>|<)\s*\$\{?(\w+)\}?/;

const mapEvaluatorType = (type: string): ThresholdOperator | null => {
  switch (type) {
    case 'gt':
      return '>';
    case 'gte':
      return '>=';
    case 'lt':
      return '<';
    case 'lte':
      return '<=';
    default:
      return null;
  }
};

/**
 * Splits a PromQL query that ends in a comparison (e.g. `... * 100 > 80`) into its
 * left-hand value expression, operator and threshold. Returns `null` when there is no
 * trailing comparison. `isBool` flags the `bool` modifier (`== bool 1`), which marks a
 * boolean up/down check rather than a value/threshold comparison.
 */
export const parseComparison = (expr?: string): ParsedComparison | null => {
  if (!expr) {
    return null;
  }

  const match = expr.match(TRAILING_COMPARISON);
  if (!match) {
    return null;
  }

  const threshold = parseFloat(match[4]);
  if (Number.isNaN(threshold)) {
    return null;
  }

  return {
    lhs: match[1].trim(),
    operator: match[2] as ThresholdOperator,
    threshold,
    isBool: Boolean(match[3]),
  };
};

/**
 * Inspects a Grafana-managed rule definition and works out how to obtain a
 * value/threshold pair via POST /v1/eval. Returns `null` for rules that have no
 * meaningful value/threshold (boolean up/down checks, unsupported expressions).
 */
export const resolveEvalPlan = (
  def: GrafanaAlertRuleDefinition
): EvalPlan | null => {
  const condition = def.data?.find((query) => query.refId === def.condition);
  if (!condition) {
    return null;
  }

  // Expression node (math / threshold / classic_conditions).
  if (condition.datasourceUid === EXPR_DATASOURCE) {
    const model = condition.model;

    if (model.type === 'math' && model.expression) {
      const match = model.expression.match(MATH_COMPARISON);
      if (!match) {
        return null;
      }
      return {
        data: def.data,
        valueRefId: match[1],
        operator: match[2] as ThresholdOperator,
        thresholdRefId: match[3],
      };
    }

    if (
      (model.type === 'threshold' || model.type === 'classic_conditions') &&
      model.conditions?.length
    ) {
      const { evaluator, query } = model.conditions[0];
      const operator = mapEvaluatorType(evaluator.type);
      const valueRefId =
        model.type === 'threshold'
          ? typeof model.expression === 'string'
            ? model.expression
            : undefined
          : query?.params?.[0];

      if (!operator || !valueRefId) {
        return null;
      }

      return {
        data: def.data,
        valueRefId,
        operator,
        thresholdConst: evaluator.params?.[0],
      };
    }

    return null;
  }

  // Datasource query with the comparison baked into PromQL.
  const parsed = parseComparison(condition.model.expr);
  if (!parsed || parsed.isBool) {
    return null;
  }

  return {
    data: [
      {
        ...condition,
        model: {
          ...condition.model,
          expr: parsed.lhs,
          instant: true,
          range: false,
        },
      },
    ],
    valueRefId: condition.refId,
    operator: parsed.operator,
    thresholdConst: parsed.threshold,
  };
};

const frameLabels = (frame: AlertEvalFrame): Record<string, string> =>
  frame.schema.fields.find((field) => field.labels)?.labels ?? {};

const frameValue = (frame: AlertEvalFrame): number | null => {
  const value = frame.data.values?.[0]?.[0];
  return typeof value === 'number' ? value : null;
};

/**
 * Picks the evaluated number for the series that matches the alert instance. Eval can
 * return one frame per series (multi-dimensional rules); the right frame is the one
 * whose labels are all present and equal in the alert's labels. Falls back to the sole
 * frame when there is only one (e.g. a constant threshold).
 */
export const pickSeriesValue = (
  frames: AlertEvalFrame[] | undefined,
  alertLabels: Record<string, string>
): number | null => {
  if (!frames?.length) {
    return null;
  }

  if (frames.length === 1) {
    return frameValue(frames[0]);
  }

  const match = frames.find((frame) =>
    Object.entries(frameLabels(frame)).every(
      ([key, value]) => alertLabels[key] === value
    )
  );

  return match ? frameValue(match) : null;
};

// `direction` and `percent` describe where the value actually sits relative to the
// threshold (positional), not the rule's alerting operator — so a `>` rule whose metric
// has dropped back below the threshold reads as "under", not a misleading "over".
export const computeValueThreshold = (
  value: number,
  threshold: number,
  operator: ThresholdOperator
): ValueThresholdResult => ({
  value,
  threshold,
  operator,
  direction: value >= threshold ? 'over' : 'under',
  percent:
    threshold === 0
      ? null
      : Math.floor((Math.abs(value - threshold) / Math.abs(threshold)) * 100),
});
