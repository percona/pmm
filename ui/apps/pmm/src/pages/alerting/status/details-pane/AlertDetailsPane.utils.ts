import { useGrafanaRulerRule } from 'hooks/api/useGrafanaRuler';
import { AlertDetailsPane } from './AlertDetailsPane.types';
import { AlertRow } from '../AlertsPage.types';

// Pretty-printed rule + alert JSON for the raw-data tab. Built here for the single
// selected alert instead of in flattenAlertRules, which runs for every alert on
// every poll.
const buildRawJson = (alert: AlertRow): string =>
  JSON.stringify(
    {
      rule: alert.rule && {
        name: alert.rule.name,
        query: alert.rule.query,
        duration: alert.rule.duration,
        labels: alert.rule.labels,
        annotations: alert.rule.annotations,
        health: alert.rule.health,
        lastError: alert.rule.lastError,
        type: alert.rule.type,
        state: alert.rule.state,
      },
      alert: alert.rawAlert,
    },
    null,
    2
  );

export const useAlertDetailsPane = (
  alert?: AlertRow
): AlertDetailsPane | null => {
  const ruleUid = alert?.rule?.uid;
  const { data: grafanaRule, isLoading } = useGrafanaRulerRule(ruleUid || '', {
    enabled: !!ruleUid,
  });

  if (!alert) {
    return null;
  }

  return {
    expression: alert.expression,
    summary: {
      alertName: alert.alertName,
      state: alert.state,
      // Pair the duration with the state shown: how long silenced when silenced,
      // otherwise how long in its current alert state.
      stateDuration: alert.silenced ? alert.silencedAge : alert.age,
      nodeName: alert.labels['node_name'],
      serviceName: alert.labels['service_name'],
      triggeredAt: alert.activeAt,
      severity: alert.labels['severity'],
      summary: alert.summary,
      description: alert.annotations['description'],
      silenced: alert.silenced,
    },
    ruleConfiguration: {
      isLoading,
      definition: grafanaRule?.grafana_alert,
      // Evaluation interval in seconds. `grafanaRule.for` is the pending period,
      // which is already shown separately as `pendingPeriod`.
      evaluationIntervalSeconds:
        grafanaRule?.grafana_alert.intervalSeconds ?? alert.ruleGroup?.interval,
      lastEvaluation: alert.ruleGroup?.lastEvaluation,
      lastEvaluationDuration: alert.ruleGroup?.evaluationTime,
      pendingPeriod: alert.rule?.duration,
      keepFiringFor: grafanaRule?.keep_firing_for,
      ruleUid: grafanaRule?.grafana_alert.uid || ruleUid,
      ruleType: alert.rule?.type,
      lastUpdated: grafanaRule?.grafana_alert.updated,
      lastUpdatedBy: grafanaRule?.grafana_alert.updated_by?.name,
      templateName: alert.rule?.labels?.['template_name'],
      folder: alert.ruleGroup?.file,
      ruleHealth: alert.rule?.health,
    },
    rawData: {
      labels: alert.labels,
      json: buildRawJson(alert),
    },
  };
};
