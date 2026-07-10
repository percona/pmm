import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { evalAlertQueries, getAlertRuleDefinition } from 'api/alerting';
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
// or when the Grafana provisioning/eval endpoints are unavailable, so the caller can
// simply hide the field.
const fetchValueThreshold = async (
  uid: string,
  labels: GrafanaRulerLabels
): Promise<ValueThresholdResult | null> => {
  if (!uid) {
    return null;
  }

  try {
    const definition = await getAlertRuleDefinition(uid);
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
  uid: string,
  labels: GrafanaRulerLabels,
  options?: Partial<UseQueryOptions<ValueThresholdResult | null>>
) =>
  useQuery({
    queryKey: [ALERT_VALUE_THRESHOLD_QUERY_KEY, uid, labels],
    queryFn: () => fetchValueThreshold(uid, labels),
    enabled: !!uid,
    staleTime: 30_000,
    ...options,
  });
