export type PrometheusAlertState = 'firing' | 'pending' | 'inactive';

export type AlertStatus =
  | 'Alerting'
  | 'Normal'
  | 'Pending'
  | 'NoData'
  | 'Error';

export interface PrometheusAlertItem {
  labels: Record<string, string>;
  annotations: Record<string, string>;
  state?: AlertStatus | PrometheusAlertState;
  activeAt?: string;
  value?: string;
}

export interface PrometheusAlertRuleItem {
  uid?: string;
  name: string;
  query?: string;
  duration?: number;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  alerts: PrometheusAlertItem[];
  health?: string;
  lastError?: string;
  type?: string;
  state?: PrometheusAlertState;
}

export interface PrometheusAlertRuleGroup {
  uid?: string;
  name?: string;
  file?: string;
  interval?: number;
  rules: PrometheusAlertRuleItem[];
  evaluationTime?: number;
  lastEvaluation?: string;
}

export interface PrometheusAlertRulesData {
  groups: PrometheusAlertRuleGroup[];
}

export interface PrometheusAlertRulesResponse {
  status?: string;
  data: PrometheusAlertRulesData;
}
