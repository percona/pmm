// Grafana Ruler API — GET /api/ruler/grafana/api/v1/rule/{uid}
// Response type for a single Grafana-managed alert rule.

import { GrafanaAlertQuery } from './alerting.types';

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
