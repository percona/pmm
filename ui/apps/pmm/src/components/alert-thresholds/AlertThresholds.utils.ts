import type {
  ListThresholdsResponse,
  PrometheusAlertRulesResponse,
  Threshold,
  ThresholdUpdate,
} from 'types/alerting.types';
import type {
  AlertThresholdRow,
  AlertThresholdsFormValues,
} from './AlertThresholds.types';
import { UNIT_SYMBOLS } from './AlertThresholds.constants';

export const formatUnit = (unit?: string): string =>
  (unit && UNIT_SYMBOLS[unit]) || '';

// A rule can expose several overridable params, and two rules duplicated in Grafana
// share a rule id, so neither part alone identifies a row. The index disambiguates
// the duplicate case, which the API explicitly permits.
export const thresholdRowId = (t: Threshold, index: number): string =>
  `${t.ruleId}:${t.paramName}:${index}`;

// Rule titles live in Grafana, not in the thresholds response, so they are joined
// on the identity label PMM stamps on every rule it creates.
export const getRuleTitles = (
  rulesData: PrometheusAlertRulesResponse
): Map<string, string> => {
  const titles = new Map<string, string>();

  for (const group of rulesData?.data?.groups ?? []) {
    for (const rule of group.rules ?? []) {
      const id = rule.labels?.pmm_rule_id;
      if (id) {
        titles.set(id, rule.name);
      }
    }
  }

  return titles;
};

// Rule titles are display-only decoration, so a caller that needs only row identity and
// values can omit them.
export const getRows = (
  data: ListThresholdsResponse | undefined,
  ruleTitles?: Map<string, string>
) =>
  (data?.thresholds ?? []).map((t, index) => ({
    ...t,
    // proto3 omits zero values, so absent means 0 rather than unknown.
    defaultValue: t.defaultValue ?? 0,
    effectiveValue: t.effectiveValue ?? 0,
    isOverridden: t.isOverridden ?? false,
    id: thresholdRowId(t, index),
    ruleTitle: ruleTitles?.get(t.ruleId) ?? '',
  }));

// Seeds the form: one field per row, keyed by the row id the table renders.
//
// Derived from the thresholds response alone. Rule titles come from a separate Grafana
// query that can land after the table is editable, and that re-polls every few seconds
// while the alerts page is mounted; were they on this path, every rules response would
// re-seed the form over whatever the operator had typed. Routed through getRows so row
// ids and zero-coercion stay defined in one place.
export const getInitialValues = (
  data: ListThresholdsResponse | undefined
): AlertThresholdsFormValues =>
  getRows(data).reduce((acc, row) => {
    acc[row.id] = row.effectiveValue;
    return acc;
  }, {} as AlertThresholdsFormValues);

// Turns the submitted form into the smallest set of changes that expresses it.
//
// Emptying a field, or typing the default back in, returns the target to the rule
// default. That is only a change when an override exists today, and it is expressed by
// omitting `value` - writing the default as an override instead would pin the target to
// today's default and stop it following a later change to the rule.
export const buildThresholdUpdates = (
  rows: AlertThresholdRow[],
  values: AlertThresholdsFormValues,
  scope: ThresholdUpdate['scope'],
  target: string
): ThresholdUpdate[] => {
  const updates: ThresholdUpdate[] = [];

  for (const row of rows) {
    const raw = values[row.id];
    const parsed =
      raw === undefined || (raw as unknown) === '' ? undefined : Number(raw);
    const cleared = parsed === undefined || Number.isNaN(parsed);

    const base = {
      scope,
      target,
      ruleId: row.ruleId,
      paramName: row.paramName,
    };

    if (cleared || parsed === row.defaultValue) {
      if (row.isOverridden) {
        updates.push(base);
      }
      continue;
    }

    if (parsed !== row.effectiveValue) {
      updates.push({ ...base, value: parsed });
    }
  }

  return updates;
};
