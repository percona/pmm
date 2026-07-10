import { AlertStatus } from 'types/alerting.types';
import { GrafanaRulerLabels } from 'types/ruler.types';

export interface AlertDetailsPane {
  expression: string;
  summary: {
    alertName: string;
    state: AlertStatus;
    stateDuration: string;
    nodeName: string;
    serviceName: string;
    triggeredAt?: string;
    severity: string;
    summary: string;
    description: string;
  };
  ruleConfiguration: {
    evaluate?: string;
    lastEvaluation?: string;
    lastEvaluationDuration?: number;
    pendingPeriod?: number;
    keepFiringFor?: string;
    ruleUid: string;
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
