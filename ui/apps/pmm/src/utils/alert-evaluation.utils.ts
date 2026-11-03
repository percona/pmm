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
  // Whether the rule's operator is satisfied at the current value (`>` rule => value > threshold,
  // `<` rule => value < threshold). Drives the gauge colour; `direction` only describes position.
  breaching: boolean;
  percent: number | null;
}

const EXPR_DATASOURCE = '__expr__';

// Trailing PromQL comparison: `<lhs> <op> [bool] <number>` at the end of the query.
const TRAILING_COMPARISON =
  /^([\s\S]*?)\s*(>=|<=|==|!=|>|<)\s*(bool\s+)?(-?\d+(?:\.\d+)?)\s*$/;

// Top-level logical operators (`and`/`or`/`unless`) — PromQL requires surrounding whitespace, so
// this won't match metric names like `node_memory_..._and_...` or `command`.
const LOGICAL_OPERATOR = /\s+(and|or|unless)\s+/i;

// `$A > $B` style condition inside a Grafana math expression.
const MATH_COMPARISON = /\$\{?(\w+)\}?\s*(>=|<=|==|!=|>|<)\s*\$\{?(\w+)\}?/;

// `$A > 80` style condition — a $ref compared against a numeric literal. Anchored so
// compound expressions (`$A > 80 || $B > 90`) don't latch onto one sub-comparison.
const MATH_CONST_COMPARISON =
  /^\s*\$\{?(\w+)\}?\s*(>=|<=|==|!=|>|<)\s*(-?\d+(?:\.\d+)?)\s*$/;

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
 * Decides whether a parsed PromQL comparison yields a meaningful value/threshold gauge. Rejects
 * equality/up-down checks (`== bool 0`, `!= ...`) and logical composites (`... and ...`, `... or ...`),
 * whose trailing comparison would latch onto a meaningless sub-comparison. `bool`-modified
 * inequalities (e.g. `... * 100 > bool 90`) ARE gauge-able — `bool` only changes the result shape.
 */
const isGaugeableComparison = (
  parsed: ParsedComparison,
  expr: string
): boolean => {
  if (parsed.operator === '==' || parsed.operator === '!=') {
    return false;
  }
  return !LOGICAL_OPERATOR.test(expr);
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
      const refMatch = model.expression.match(MATH_COMPARISON);
      if (refMatch) {
        return {
          data: def.data,
          valueRefId: refMatch[1],
          operator: refMatch[2] as ThresholdOperator,
          thresholdRefId: refMatch[3],
        };
      }

      const constMatch = model.expression.match(MATH_CONST_COMPARISON);
      if (constMatch) {
        const threshold = parseFloat(constMatch[3]);
        if (!Number.isNaN(threshold)) {
          return {
            data: def.data,
            valueRefId: constMatch[1],
            operator: constMatch[2] as ThresholdOperator,
            thresholdConst: threshold,
          };
        }
      }

      return null;
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
  const expr = condition.model.expr;
  const parsed = parseComparison(expr);
  if (!parsed || !expr || !isGaugeableComparison(parsed, expr)) {
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

const getValueFieldIndex = (frame: AlertEvalFrame): number => {
  const numberFieldIndex = frame.schema.fields.findIndex(
    (field) => field.type === 'number'
  );
  if (numberFieldIndex >= 0) {
    return numberFieldIndex;
  }

  const labelledFieldIndex = frame.schema.fields.findIndex(
    (field) => field.labels
  );
  if (labelledFieldIndex >= 0) {
    return labelledFieldIndex;
  }

  return frame.schema.fields.length === 1 ? 0 : -1;
};

const frameLabels = (frame: AlertEvalFrame): Record<string, string> => {
  const valueFieldIndex = getValueFieldIndex(frame);
  return frame.schema.fields[valueFieldIndex]?.labels ?? {};
};

// Reads the newest sample: range queries (classic conditions, math over range data)
// return the whole window, and the last sample is the current value. The classic-condition
// reducer (avg/max/...) is NOT applied — the gauge is a best-effort current-value display.
const frameValue = (frame: AlertEvalFrame): number | null => {
  const valueFieldIndex = getValueFieldIndex(frame);
  const samples = frame.data.values?.[valueFieldIndex];
  const value = samples?.length ? samples[samples.length - 1] : undefined;
  return typeof value === 'number' ? value : null;
};

/**
 * Picks the evaluated number for the series that matches the alert instance. Eval can
 * return one frame per series (multi-dimensional rules); the right frame is the one
 * whose labels are all present and equal in the alert's labels. Label-less frames
 * (constants, expression results) match trivially. A sole frame with mismatched labels
 * is rejected — it belongs to a different alert instance — unless `allowSoleFrame` is
 * set, which threshold lookups use because thresholds are typically scalar.
 */
export const pickSeriesValue = (
  frames: AlertEvalFrame[] | undefined,
  alertLabels: Record<string, string>,
  { allowSoleFrame = false }: { allowSoleFrame?: boolean } = {}
): number | null => {
  if (!frames?.length) {
    return null;
  }

  const match = frames.find((frame) =>
    Object.entries(frameLabels(frame)).every(
      ([key, value]) => alertLabels[key] === value
    )
  );

  if (match) {
    return frameValue(match);
  }

  return allowSoleFrame && frames.length === 1 ? frameValue(frames[0]) : null;
};

// Whether the rule's operator holds at the current value (i.e. the rule is breaching).
const isBreaching = (
  value: number,
  threshold: number,
  operator: ThresholdOperator
): boolean => {
  switch (operator) {
    case '>':
      return value > threshold;
    case '>=':
      return value >= threshold;
    case '<':
      return value < threshold;
    case '<=':
      return value <= threshold;
    case '==':
      return value === threshold;
    case '!=':
      return value !== threshold;
  }
};

// `direction` and `percent` describe where the value sits relative to the threshold (positional),
// while `breaching` reflects the rule's alerting operator (a `<` rule breaches when the value is
// below the threshold) and drives the gauge colour.
export const computeValueThreshold = (
  value: number,
  threshold: number,
  operator: ThresholdOperator
): ValueThresholdResult => ({
  value,
  threshold,
  operator,
  direction: value >= threshold ? 'over' : 'under',
  breaching: isBreaching(value, threshold, operator),
  percent:
    threshold === 0
      ? null
      : Math.floor((Math.abs(value - threshold) / Math.abs(threshold)) * 100),
});
