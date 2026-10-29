import {
  AlertStatus,
  PrometheusAlertItem,
  PrometheusAlertRuleGroup,
  PrometheusAlertRuleItem,
} from 'types/alerting.types';

export interface AlertRow {
  type: 'alert';
  id: string;
  alertName: string;
  ruleName: string;
  ruleGroupUid?: string;
  ruleGroup?: PrometheusAlertRuleGroup;
  rule?: PrometheusAlertRuleItem;
  // The raw alert item as returned by the API, for the raw-data tab.
  rawAlert: PrometheusAlertItem;
  state: AlertStatus;
  nodeId: string;
  serviceName: string;
  summary: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  expression: string;
  activeAt?: string;
  age: string;
  // Whether an Alertmanager silence currently suppresses this alert, the ids of
  // the silences doing so (empty when not silenced), and how long it has been
  // silenced (time since the earliest governing silence started; '-' if unknown).
  silenced: boolean;
  silencedBy: string[];
  silencedAge: string;
}

// Per-alert silence data, keyed by visible-label signature in the silence map.
export interface SilenceInfo {
  silencedBy: string[];
  // ISO start time of the earliest silence suppressing the alert.
  silencedSince?: string;
}

export interface NodeGroupRow {
  type: 'node';
  id: string;
  nodeId: string;
  state: AlertStatus;
  alertCount: number;
  alerts: AlertRow[];
}

export type AlertsTableRow = AlertRow | NodeGroupRow;
