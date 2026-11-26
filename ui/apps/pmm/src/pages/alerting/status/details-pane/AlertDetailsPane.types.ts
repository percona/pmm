import {
  AlertStatus,
  GrafanaAlertRuleDefinition,
  GrafanaRulerLabels,
} from 'types/alerting.types';

export interface AlertDetailsPane {
  expression: string;
  summary: {
    alertName: string;
    state: AlertStatus;
    stateDuration: string;
    nodeName?: string;
    serviceName?: string;
    triggeredAt?: string;
    severity?: string;
    summary: string;
    description?: string;
    silenced: boolean;
  };
  ruleConfiguration: {
    isLoading: boolean;
    definition?: GrafanaAlertRuleDefinition;
    evaluationIntervalSeconds?: number;
    lastEvaluation?: string;
    lastEvaluationDuration?: number;
    pendingPeriod?: number;
    keepFiringFor?: string;
    ruleUid?: string;
    ruleType?: string;
    lastUpdated?: string;
    lastUpdatedBy?: string;
    templateName?: string;
    folder?: string;
    ruleHealth?: string;
  };
  rawData: {
    labels: GrafanaRulerLabels;
    json: string;
  };
}
