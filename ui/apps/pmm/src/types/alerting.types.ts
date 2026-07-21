export type PrometheusAlertState =
  | 'firing'
  | 'pending'
  | 'recovering'
  | 'inactive';

export type AlertStatus =
  | 'Alerting'
  | 'Normal'
  | 'Pending'
  | 'Recovering'
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

// Alertmanager v2 alert. Silence/suppression only exists here, not in the
// Prometheus rules API: a suppressed alert has `status.state === 'suppressed'`
// and a non-empty `status.silencedBy`. Fields stay camelCase because `grafanaApi`
// does not run the case-converter middleware.
export type AlertmanagerAlertState = 'unprocessed' | 'active' | 'suppressed';

export interface AlertmanagerAlert {
  labels: Record<string, string>;
  annotations: Record<string, string>;
  startsAt: string;
  endsAt: string;
  updatedAt: string;
  fingerprint: string;
  status: {
    state: AlertmanagerAlertState;
    silencedBy: string[];
    inhibitedBy: string[];
  };
}

export type AlertmanagerSilenceState = 'expired' | 'active' | 'pending';

// A silence from the Alertmanager. `startsAt` is when the silence began — joined
// to an alert's `status.silencedBy` id, it lets us compute how long an alert has
// been silenced.
export interface AlertmanagerSilence {
  id: string;
  startsAt: string;
  endsAt: string;
  updatedAt: string;
  comment?: string;
  createdBy?: string;
  status: {
    state: AlertmanagerSilenceState;
  };
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
    fields: {
      name?: string;
      type?: 'time' | 'number' | 'string' | 'boolean' | string;
      labels?: Record<string, string>;
    }[];
  };
  data: { values: unknown[][] };
}

export interface AlertEvalResult {
  status: number;
  frames: AlertEvalFrame[];
}

export interface AlertEvalResponse {
  results: Record<string, AlertEvalResult>;
}

export type AlertSeverity =
  | 'unknown'
  | 'emergency'
  | 'alert'
  | 'critical'
  | 'error'
  | 'warning'
  | 'notice'
  | 'info'
  | 'debug';

export type GrafanaRulerLabels = Record<string, string>;
export type GrafanaRulerAnnotations = Record<string, string>;

export enum GrafanaAlertStateDecision {
  Alerting = 'Alerting',
  NoData = 'NoData',
  KeepLast = 'KeepLast',
  OK = 'OK',
  Error = 'Error',
}

export interface GrafanaRulerNotificationSettings {
  receiver: string;
  group_by?: string[];
  group_wait?: string;
  group_interval?: string;
  repeat_interval?: string;
  mute_time_intervals?: string[];
  active_time_intervals?: string[];
}

export interface GrafanaRulerEditorSettings {
  simplified_query_and_expressions_section: boolean;
  simplified_notifications_section: boolean;
}

export interface GrafanaRulerUpdatedBy {
  uid: string;
  name: string;
}

export interface GrafanaRulerRuleDefinition {
  uid: string;
  title: string;
  condition: string;
  no_data_state?: GrafanaAlertStateDecision;
  exec_err_state?: GrafanaAlertStateDecision;
  data: GrafanaAlertQuery[];
  is_paused?: boolean;
  notification_settings?: GrafanaRulerNotificationSettings;
  metadata?: {
    editor_settings?: GrafanaRulerEditorSettings;
  };
  record?: {
    metric: string;
    from: string;
    target_datasource_uid?: string;
  };
  intervalSeconds?: number;
  missing_series_evals_to_resolve?: number;
  id?: string;
  guid?: string;
  namespace_uid: string;
  rule_group: string;
  provenance?: string;
  updated?: string;
  updated_by?: GrafanaRulerUpdatedBy | null;
  version?: number;
  message?: string;
}

export interface GrafanaRulerRuleDTO {
  grafana_alert: GrafanaRulerRuleDefinition;
  for?: string;
  keep_firing_for?: string;
  annotations?: GrafanaRulerAnnotations;
  labels?: GrafanaRulerLabels;
}
