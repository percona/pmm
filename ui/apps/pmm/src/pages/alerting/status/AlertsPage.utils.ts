import {
  AlertStatus,
  AlertmanagerAlert,
  AlertmanagerSilence,
  PrometheusAlertItem,
  PrometheusAlertRuleGroup,
  PrometheusAlertRuleItem,
  PrometheusAlertState,
  PrometheusAlertRulesResponse,
} from 'types/alerting.types';
import { AlertRow, NodeGroupRow, SilenceInfo } from './AlertsPage.types';
import { formatDurationSeconds } from 'utils/duration.utils';
import { isPrivateLabelKey } from 'utils/alerting.utils';

const NODE_NAME_LABEL = 'node_name';
const UNKNOWN_NODE = 'unknown-node';

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

// Stable key over an alert's visible (non-private) labels, used to correlate a
// rules-API alert instance with an Alertmanager alert. Silence matchers only use
// visible labels, so private-label drift between the two APIs is ignored here.
export const serializeVisibleLabels = (
  labels: Record<string, string>
): string => {
  const visible = Object.entries(labels)
    .filter(([key]) => !isPrivateLabelKey(key))
    .sort(([a], [b]) => a.localeCompare(b));

  return JSON.stringify(visible);
};

// Map of visible-label key -> silence info, built by correlating the suppressed
// Alertmanager alerts with the configured silences: each entry carries the silence
// ids and the earliest silence start, so flattenAlertRules can flag matching rows
// as silenced and compute how long they have been silenced.
export const buildSilenceMap = (
  alerts?: AlertmanagerAlert[],
  silences?: AlertmanagerSilence[]
): Map<string, SilenceInfo> => {
  const silenceStartsById = new Map<string, string>();
  for (const silence of silences || []) {
    silenceStartsById.set(silence.id, silence.startsAt);
  }

  const map = new Map<string, SilenceInfo>();

  for (const alert of alerts || []) {
    if (alert.status.state === 'suppressed' && alert.status.silencedBy.length) {
      const silencedBy = alert.status.silencedBy;
      // ISO timestamps sort chronologically as strings, so the first is earliest.
      const silencedSince = silencedBy
        .map((id) => silenceStartsById.get(id))
        .filter((start): start is string => !!start)
        .sort()[0];

      map.set(serializeVisibleLabels(alert.labels), {
        silencedBy,
        silencedSince,
      });
    }
  }

  return map;
};

export const flattenAlertRules = (
  data?: PrometheusAlertRulesResponse,
  silenceMap?: Map<string, SilenceInfo>
): AlertRow[] => {
  if (!data?.data.groups.length) {
    return [];
  }

  const rows = data.data.groups.flatMap((group) =>
    (group.rules || []).flatMap((rule) =>
      (rule.alerts || []).map((alert) => {
        const silence = silenceMap?.get(serializeVisibleLabels(alert.labels));
        const silencedBy = silence?.silencedBy || [];

        return {
          type: 'alert' as const,
          id: getAlertId(group, rule, alert.labels),
          alertName: getAlertName(alert, rule),
          ruleName: rule.name || 'Unnamed rule',
          ruleGroupUid: rule.uid,
          ruleGroup: group,
          rule,
          rawAlert: alert,
          state: resolveState(alert, rule),
          nodeId: getAlertNodeId(alert),
          serviceName: getAlertServiceName(alert),
          summary: getSummary(alert),
          labels: alert.labels,
          annotations: alert.annotations,
          expression: rule.query || '',
          activeAt: alert.activeAt,
          age: getAge(alert.activeAt),
          silenced: silencedBy.length > 0,
          silencedBy,
          silencedAge: getAge(silence?.silencedSince),
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
