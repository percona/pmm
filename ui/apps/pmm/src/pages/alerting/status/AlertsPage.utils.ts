import {
  AlertStatus,
  PrometheusAlertItem,
  PrometheusAlertRuleGroup,
  PrometheusAlertRuleItem,
  PrometheusAlertState,
  PrometheusAlertRulesResponse,
} from 'types/alerting.types';
import { AlertRow, AlertsTableRow, NodeGroupRow } from './AlertsPage.types';
import { formatDurationSeconds } from 'utils/duration.utils';

const NODE_NAME_LABEL = 'node_name';
const UNKNOWN_NODE = 'unknown-node';

export const findAlertTableRowById = (
  rows: AlertsTableRow[],
  id?: string
): AlertsTableRow | undefined => {
  if (!id) {
    return undefined;
  }

  for (const row of rows) {
    if (row.id === id) {
      return row;
    }

    if (row.type === 'node') {
      const alert = row.alerts.find((candidate) => candidate.id === id);
      if (alert) {
        return alert;
      }
    }
  }

  return undefined;
};

const GROUP_STATE_PRIORITY: Record<AlertStatus, number> = {
  Alerting: 5,
  Error: 4,
  Recovering: 3,
  Pending: 2,
  NoData: 1,
  Normal: 0,
};

const ALERT_STATUSES = new Set<AlertStatus>([
  'Alerting',
  'Error',
  'Recovering',
  'Pending',
  'NoData',
  'Normal',
]);

const isAlertStatus = (state: string): state is AlertStatus =>
  ALERT_STATUSES.has(state as AlertStatus);

const mapRuleStateToAlertState = (
  ruleState?: PrometheusAlertState
): AlertStatus => {
  switch (ruleState) {
    case 'firing':
      return 'Alerting';
    case 'pending':
      return 'Pending';
    case 'recovering':
      return 'Recovering';
    case 'inactive':
      return 'Normal';
    default:
      return 'NoData';
  }
};

const resolveState = (
  alert: PrometheusAlertItem,
  rule: PrometheusAlertRuleItem
): AlertStatus => {
  if (!alert.state) {
    return mapRuleStateToAlertState(rule.state);
  }

  return isAlertStatus(alert.state)
    ? alert.state
    : mapRuleStateToAlertState(alert.state);
};

const getAge = (activeAt?: string): string => {
  if (!activeAt) {
    return '-';
  }

  const timestamp = new Date(activeAt).getTime();

  if (Number.isNaN(timestamp)) {
    return '-';
  }

  const diffMs = Math.max(Date.now() - timestamp, 0);
  return formatDurationSeconds(diffMs / 1000);
};

const getSource = (labels: Record<string, string>) =>
  labels.service_name || labels.service || labels.job || labels.instance || '-';

const getServiceName = (labels: Record<string, string>) =>
  labels.service_name || labels.service || '-';

const getSummary = (alert: PrometheusAlertItem) =>
  alert.annotations.summary || alert.annotations.description || '-';

const getNodeName = (labels: Record<string, string>) =>
  labels[NODE_NAME_LABEL] || UNKNOWN_NODE;

const getAlertName = (
  alert: PrometheusAlertItem,
  rule: PrometheusAlertRuleItem
) => alert.labels.alertname || rule.name || 'Unnamed alert';

const getAlertNodeId = (alert: PrometheusAlertItem): string => {
  const nodeId = getNodeName(alert.labels);
  return nodeId === UNKNOWN_NODE ? '' : nodeId;
};

const getAlertServiceName = (alert: PrometheusAlertItem): string => {
  const serviceName = getServiceName(alert.labels);
  return serviceName === '-' ? '' : serviceName;
};

const getAlertId = (
  group: PrometheusAlertRuleGroup,
  rule: PrometheusAlertRuleItem,
  labels: Record<string, string>
): string => {
  const sortedLabels = Object.entries(labels).sort(([a], [b]) =>
    a.localeCompare(b)
  );

  const ruleId =
    rule.uid || `${group.uid || group.name || 'group'}:${rule.name}`;

  return `${ruleId}:${JSON.stringify(sortedLabels)}`;
};

export const flattenAlertRules = (
  data?: PrometheusAlertRulesResponse
): AlertRow[] => {
  if (!data?.data.groups.length) {
    return [];
  }

  const rows = data.data.groups.flatMap((group) =>
    (group.rules || []).flatMap((rule) =>
      (rule.alerts || []).map((alert) => {
        const ruleDetails = {
          name: rule.name,
          query: rule.query,
          duration: rule.duration,
          labels: rule.labels,
          annotations: rule.annotations,
          health: rule.health,
          lastError: rule.lastError,
          type: rule.type,
          state: rule.state,
        };

        return {
          type: 'alert' as const,
          id: getAlertId(group, rule, alert.labels),
          alertName: getAlertName(alert, rule),
          ruleName: rule.name || 'Unnamed rule',
          ruleGroupUid: rule.uid,
          ruleGroup: group,
          rule,
          state: resolveState(alert, rule),
          nodeId: getAlertNodeId(alert),
          serviceName: getAlertServiceName(alert),
          summary: getSummary(alert),
          source: getSource(alert.labels),
          labels: alert.labels,
          annotations: alert.annotations,
          expression: rule.query || '',
          value: alert.value,
          activeAt: alert.activeAt,
          age: getAge(alert.activeAt),
          rawJson: JSON.stringify(
            {
              rule: ruleDetails,
              alert,
            },
            null,
            2
          ),
        };
      })
    )
  );

  return rows.sort((a, b) =>
    `${a.ruleName}:${a.alertName}`.localeCompare(`${b.ruleName}:${b.alertName}`)
  );
};

export const groupAlertsByNode = (rows: AlertRow[]): NodeGroupRow[] => {
  const grouped = new Map<string, AlertRow[]>();

  for (const row of rows) {
    const nodeId = row.nodeId || UNKNOWN_NODE;
    const nodeRows = grouped.get(nodeId) || [];
    nodeRows.push(row);
    grouped.set(nodeId, nodeRows);
  }

  const groupedRows = [...grouped.entries()]
    .map(([nodeId, alerts]) => {
      const state = alerts
        .map((alert) => alert.state)
        .sort((a, b) => GROUP_STATE_PRIORITY[b] - GROUP_STATE_PRIORITY[a])[0];

      return {
        type: 'node' as const,
        id: `node:${nodeId}`,
        nodeId,
        state,
        alertCount: alerts.length,
        alerts: alerts.sort((a, b) =>
          `${a.ruleName}:${a.alertName}`.localeCompare(
            `${b.ruleName}:${b.alertName}`
          )
        ),
      };
    })
    .sort((a, b) => a.nodeId.localeCompare(b.nodeId));

  return groupedRows;
};
