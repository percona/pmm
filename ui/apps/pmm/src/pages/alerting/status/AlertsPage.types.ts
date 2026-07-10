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
