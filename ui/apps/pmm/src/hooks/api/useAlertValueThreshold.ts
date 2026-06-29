import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { evalAlertQueries, getAlertRuleDefinition } from 'api/alerting';
import { AlertRow } from 'pages/alerting/status/AlertsPage.types';
import {
  computeValueThreshold,
  pickSeriesValue,
  resolveEvalPlan,
  ValueThresholdResult,
} from 'pages/alerting/status/details-pane/alertEvaluation.utils';

export const ALERT_VALUE_THRESHOLD_QUERY_KEY = 'alerting:valueThreshold';

// Re-evaluates an alert's rule to derive the actual value/threshold pair shown in the
// detail view. Returns `null` (rather than throwing) for rules with no value/threshold
// or when the Grafana provisioning/eval endpoints are unavailable, so the caller can
// simply hide the field.
const fetchValueThreshold = async (
  alert: AlertRow
): Promise<ValueThresholdResult | null> => {
  const uid = alert.rule?.uid;
  if (!uid) {
    return null;
  }

  try {
    // TODO(PMM-14911): the provisioning API (getAlertRuleDefinition) is typically
    // editor/admin-gated, so Viewer users get 401/403 and the field silently hides.
    // If viewers need the value/threshold, switch to the `/rules`-string fallback:
    // parse the value-expr + threshold from the already-fetched rule `query`
    // (single-query: `<expr> <op> <thr>`, multi-expr: `<expr> | <thr>`) plus
    // `queriedDatasourceUIDs`, then eval only the value-expr. Verify the required
    // role on a Viewer account before relying on this path.
    const definition = await getAlertRuleDefinition(uid);
    const plan = resolveEvalPlan(definition);
    if (!plan) {
      return null;
    }

    const { results } = await evalAlertQueries(plan.data);

    const value = pickSeriesValue(
      results[plan.valueRefId]?.frames,
      alert.labels
    );
    if (value === null) {
      return null;
    }

    const threshold =
      plan.thresholdConst ??
      (plan.thresholdRefId
        ? pickSeriesValue(results[plan.thresholdRefId]?.frames, alert.labels)
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
  alert: AlertRow,
  options?: Partial<UseQueryOptions<ValueThresholdResult | null>>
) =>
  useQuery({
    queryKey: [ALERT_VALUE_THRESHOLD_QUERY_KEY, alert.rule?.uid, alert.id],
    queryFn: () => fetchValueThreshold(alert),
    enabled: !!alert.rule?.uid,
    staleTime: 30_000,
    ...options,
  });
