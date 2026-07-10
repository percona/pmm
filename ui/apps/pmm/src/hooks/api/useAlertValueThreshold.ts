import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { evalAlertQueries } from 'api/alerting';
import { GrafanaAlertRuleDefinition } from 'types/alerting.types';
import { GrafanaRulerLabels } from 'types/ruler.types';
import {
  computeValueThreshold,
  pickSeriesValue,
  resolveEvalPlan,
  ValueThresholdResult,
} from 'utils/alert-evaluation.utils';

export const ALERT_VALUE_THRESHOLD_QUERY_KEY = 'alerting:valueThreshold';

// Re-evaluates an alert's rule to derive the actual value/threshold pair shown in the
// detail view. Returns `null` (rather than throwing) for rules with no value/threshold
// or when Grafana's eval endpoint is unavailable, so the caller can simply hide the
// field.
const fetchValueThreshold = async (
  definition: GrafanaAlertRuleDefinition,
  labels: GrafanaRulerLabels
): Promise<ValueThresholdResult | null> => {
  try {
    const plan = resolveEvalPlan(definition);
    if (!plan) {
      return null;
    }

    const { results } = await evalAlertQueries(plan.data);

    const value = pickSeriesValue(results[plan.valueRefId]?.frames, labels);
    if (value === null) {
      return null;
    }

    const threshold =
      plan.thresholdConst ??
      (plan.thresholdRefId
        ? pickSeriesValue(results[plan.thresholdRefId]?.frames, labels)
        : null);
    if (threshold === null || threshold === undefined) {
      return null;
    }

    return computeValueThreshold(value, threshold, plan.operator);
  } catch {
    // Missing permissions / network errors degrade to a hidden field.
    return null;
  }
};

export const useAlertValueThreshold = (
  definition: GrafanaAlertRuleDefinition | undefined,
  labels: GrafanaRulerLabels,
  options?: Partial<UseQueryOptions<ValueThresholdResult | null>>
) =>
  useQuery({
    queryKey: [ALERT_VALUE_THRESHOLD_QUERY_KEY, definition?.uid, labels],
    queryFn: () =>
      definition
        ? fetchValueThreshold(definition, labels)
        : Promise.resolve(null),
    enabled: !!definition,
    staleTime: 30_000,
    ...options,
  });
