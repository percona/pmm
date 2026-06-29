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
  keepFiringFor?: number;
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

// Structured Grafana-managed rule definition (from the provisioning API). The
// `data` array holds the datasource queries and `__expr__` expression nodes that
// make up the rule; `condition` is the refId whose result decides firing.
export interface GrafanaExpressionEvaluator {
  type: string; // 'gt' | 'lt' | 'within_range' | 'outside_range' | ...
  params: number[];
}

export interface GrafanaExpressionCondition {
  evaluator: GrafanaExpressionEvaluator;
  query?: { params?: string[] };
}

export interface GrafanaAlertQueryModel {
  refId: string;
  type?: string; // 'math' | 'threshold' | 'classic_conditions' | 'reduce' | ...
  expr?: string;
  expression?: string;
  conditions?: GrafanaExpressionCondition[];
  [key: string]: unknown;
}

export interface GrafanaAlertQuery {
  refId: string;
  queryType?: string;
  relativeTimeRange?: { from: number; to: number };
  datasourceUid: string;
  model: GrafanaAlertQueryModel;
}

export interface GrafanaAlertRuleDefinition {
  uid: string;
  condition: string;
  data: GrafanaAlertQuery[];
}

// Response of POST /v1/eval — evaluated values keyed by query refId.
export interface AlertEvalFrame {
  schema: {
    refId?: string;
    fields: { labels?: Record<string, string> }[];
  };
  data: { values: number[][] };
}

export interface AlertEvalResult {
  status: number;
  frames: AlertEvalFrame[];
}

export interface AlertEvalResponse {
  results: Record<string, AlertEvalResult>;
}
