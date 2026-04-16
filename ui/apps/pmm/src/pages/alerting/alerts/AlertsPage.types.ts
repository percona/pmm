import { AlertStatus } from 'types/alerting.types';

export interface AlertRow {
  type: 'alert';
  id: string;
  alertName: string;
  ruleName: string;
  state: AlertStatus;
  nodeId: string;
  serviceName: string;
  summary: string;
  source: string;
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
