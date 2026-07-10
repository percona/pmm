import { useGrafanaRulerRule } from 'hooks/api/useGrafanaRuler';
import { AlertDetailsPane } from './AlertDetailsPane.types';
import { AlertRow } from '../AlertsPage.types';

export const useAlertDetailsPane = (
  alert?: AlertRow
): AlertDetailsPane | null => {
  const { data: grafanaRule } = useGrafanaRulerRule(alert?.rule?.uid!, {
    enabled: !!alert,
  });

  if (!grafanaRule || !alert) {
    return null;
  }

  return {
    expression: alert.expression,
    summary: {
      alertName: alert.alertName,
      state: alert.state,
      stateDuration: alert.age,
      nodeName: alert.labels['node_name'],
      serviceName: alert.labels['service_name'],
      triggeredAt: alert.activeAt,
      severity: alert.labels['severity'],
      summary: alert.summary,
      description: alert.annotations['description'],
    },
    ruleConfiguration: {
      evaluate: grafanaRule.for,
      lastEvaluation: alert.ruleGroup?.lastEvaluation,
      lastEvaluationDuration: alert.ruleGroup?.evaluationTime,
      pendingPeriod: alert.rule?.duration,
      keepFiringFor: grafanaRule.keep_firing_for,
      ruleUid: grafanaRule.grafana_alert.uid,
      ruleType: alert.rule?.type,
      lastUpdated: grafanaRule.grafana_alert.updated,
      lastUpdatedBy: grafanaRule.grafana_alert.updated_by?.name,
      templateName: alert.rule?.labels?.['template_name'],
      folder: alert.ruleGroup?.file,
      ruleHealth: alert.rule?.health,
    },
    rawData: {
      labels: alert.labels,
      json: alert.rawJson,
    },
  };
};
