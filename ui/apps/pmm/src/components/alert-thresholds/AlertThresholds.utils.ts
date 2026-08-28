import type {
  ListThresholdsResponse,
  PrometheusAlertRulesResponse,
  Threshold,
} from 'types/alerting.types';
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

export const getRows = (
  data: ListThresholdsResponse | undefined,
  ruleTitles: Map<string, string>
) =>
  (data?.thresholds ?? []).map((t, index) => ({
    ...t,
    // proto3 omits zero values, so absent means 0 rather than unknown.
    defaultValue: t.defaultValue ?? 0,
    effectiveValue: t.effectiveValue ?? 0,
    isOverridden: t.isOverridden ?? false,
    id: thresholdRowId(t, index),
    ruleTitle: ruleTitles.get(t.ruleId) ?? '',
  }));
