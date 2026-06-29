import { AlertStatus, PrometheusAlertRuleGroup } from 'types/alerting.types';

export interface AlertRow {
  type: 'alert';
  id: string;
  alertName: string;
  ruleName: string;
  ruleGroupUid?: string;
  ruleGroup: PrometheusAlertRuleGroup;
  state: AlertStatus;
  nodeId: string;
  serviceName: string;
  summary: string;
  source: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  expression: string;
  value?: string;
  activeAt?: string;
  age: string;
  rawJson: string;
}

export interface NodeGroupRow {
  type: 'node';
  id: string;
  nodeId: string;
  state: AlertStatus;
  alertCount: number;
  alerts: AlertRow[];
}

export type AlertsTableRow = (AlertRow | NodeGroupRow) & {
  timezone?: string;
};
