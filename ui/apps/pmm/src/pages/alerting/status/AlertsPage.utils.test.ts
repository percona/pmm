import {
  AlertmanagerAlert,
  AlertmanagerSilence,
  PrometheusAlertRulesResponse,
} from 'types/alerting.types';
import {
  buildSilenceMap,
  findLinkedAlert,
  flattenAlertRules,
  getAlertLinkParams,
  groupAlertsByNode,
  withAlertLink,
} from './AlertsPage.utils';
import { AlertRow } from './AlertsPage.types';

type OptionalAlertRowKeys =
  | 'labels'
  | 'annotations'
  | 'expression'
  | 'rawAlert'
  | 'silenced'
  | 'silencedBy'
  | 'silencedAge';

const createAlertRow = (
  row: Omit<AlertRow, OptionalAlertRowKeys> &
    Partial<Pick<AlertRow, OptionalAlertRowKeys>>
): AlertRow => ({
  labels: {},
  annotations: {},
  expression: '',
  rawAlert: { labels: {}, annotations: {} },
  silenced: false,
  silencedBy: [],
  silencedAge: '-',
  ...row,
});

const createSuppressedAlert = (
  labels: Record<string, string>,
  silencedBy: string[]
): AlertmanagerAlert => ({
  labels,
  annotations: {},
  startsAt: '',
  endsAt: '',
  updatedAt: '',
  fingerprint: '',
  status: { state: 'suppressed', silencedBy, inhibitedBy: [] },
});

const createSilence = (id: string, startsAt: string): AlertmanagerSilence => ({
  id,
  startsAt,
  endsAt: '',
  updatedAt: '',
  status: { state: 'active' },
});

describe('flattenAlertRules', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns alert rows derived from rules and alerts', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-15T12:00:00.000Z'));

    const payload: PrometheusAlertRulesResponse = {
      data: {
        groups: [
          {
            name: 'mysql-group',
            rules: [
              {
                name: 'mysql_replication_lag',
                alerts: [
                  {
                    state: 'pending',
                    activeAt: '2026-04-15T11:50:00.000Z',
                    labels: {
                      node_name: 'node-a',
                      alertname: 'MySQL Replication Delay',
                      service_name: 'mysql-service-a',
                    },
                    annotations: {
                      summary: 'Replica lag detected',
                    },
                  },
                  {
                    state: 'firing',
                    activeAt: '2026-04-15T11:40:00.000Z',
                    labels: {
                      node_name: 'node-a',
                      alertname: 'MySQL Replication Broken',
                      service_name: 'mysql-service-a',
                    },
                    annotations: {
                      summary: 'Replica stopped',
                    },
                  },
                  {
                    state: 'inactive',
                    labels: {
                      node_name: 'node-a',
                      alertname: 'MySQL Connections',
                      service_name: 'mysql-service-b',
                    },
                    annotations: {
                      summary: 'Connections are healthy',
                    },
                  },
                ],
              },
              {
                name: 'mysql_connections',
                alerts: [
                  {
                    state: 'pending',
                    labels: {
                      node_name: 'node-b',
                      service_name: 'mysql-service-b',
                    },
                    annotations: {},
                  },
                ],
              },
              {
                name: 'rule_without_alerts',
                state: 'inactive',
                annotations: {
                  summary: 'No active alerts',
                },
                alerts: [],
              },
            ],
          },
        ],
      },
    };

    const rows = flattenAlertRules(payload);

    expect(rows).toHaveLength(4);
    const mysqlConnectionsAlert = rows.find(
      (row) => row.alertName === 'MySQL Connections'
    );
    expect(mysqlConnectionsAlert).toBeDefined();
    expect(mysqlConnectionsAlert?.type).toBe('alert');
    expect(mysqlConnectionsAlert?.ruleName).toBe('mysql_replication_lag');
    expect(mysqlConnectionsAlert?.state).toBe('Normal');
    expect(mysqlConnectionsAlert?.nodeId).toBe('node-a');
    expect(mysqlConnectionsAlert?.serviceName).toBe('mysql-service-b');
    expect(mysqlConnectionsAlert?.summary).toBe('Connections are healthy');

    const replicationBrokenAlert = rows.find(
      (row) => row.alertName === 'MySQL Replication Broken'
    );
    expect(replicationBrokenAlert).toBeDefined();
    expect(replicationBrokenAlert?.state).toBe('Alerting');
    expect(replicationBrokenAlert?.nodeId).toBe('node-a');
    expect(replicationBrokenAlert?.serviceName).toBe('mysql-service-a');
    expect(replicationBrokenAlert?.age).toBe('20m');

    const mysqlConnectionsRuleAlert = rows.find(
      (row) => row.ruleName === 'mysql_connections'
    );
    expect(mysqlConnectionsRuleAlert).toBeDefined();
    expect(mysqlConnectionsRuleAlert?.nodeId).toBe('node-b');
    expect(mysqlConnectionsRuleAlert?.serviceName).toBe('mysql-service-b');
  });

  it('carries the originating rule onto each alert row', () => {
    const payload: PrometheusAlertRulesResponse = {
      data: {
        groups: [
          {
            name: 'mysql-group',
            interval: 60,
            rules: [
              {
                uid: 'rule-uid-1',
                name: 'mysql_replication_lag',
                query: 'up == 0',
                duration: 300,
                keepFiringFor: 120,
                type: 'alerting',
                health: 'ok',
                alerts: [
                  {
                    state: 'firing',
                    value: '42',
                    labels: {
                      node_name: 'node-a',
                      alertname: 'MySQL Replication Delay',
                      service_name: 'mysql-service-a',
                    },
                    annotations: {},
                  },
                ],
              },
            ],
          },
        ],
      },
    };

    const rows = flattenAlertRules(payload);

    expect(rows).toHaveLength(1);
    expect(rows[0].rule).toBeDefined();
    expect(rows[0].rule?.uid).toBe('rule-uid-1');
    expect(rows[0].rule?.duration).toBe(300);
    expect(rows[0].rule?.keepFiringFor).toBe(120);
    expect(rows[0].rule?.type).toBe('alerting');
    expect(rows[0].rule?.health).toBe('ok');
    expect(rows[0].rawAlert.value).toBe('42');
  });

  it('creates unique stable IDs from all alert labels', () => {
    const payload: PrometheusAlertRulesResponse = {
      data: {
        groups: [
          {
            rules: [
              {
                uid: 'rule-uid-1',
                name: 'mysql_connections',
                alerts: [
                  {
                    labels: {
                      node_name: 'node-a',
                      service_name: 'mysql-service-a',
                      database: 'database-a',
                    },
                    annotations: {},
                  },
                  {
                    labels: {
                      database: 'database-b',
                      service_name: 'mysql-service-a',
                      node_name: 'node-a',
                    },
                    annotations: {},
                  },
                ],
              },
            ],
          },
        ],
      },
    };

    const rows = flattenAlertRules(payload);

    expect(rows).toHaveLength(2);
    expect(new Set(rows.map((row) => row.id))).toHaveLength(2);

    const reorderedPayload = structuredClone(payload);
    reorderedPayload.data.groups[0].rules[0].alerts[0].labels = {
      database: 'database-a',
      service_name: 'mysql-service-a',
      node_name: 'node-a',
    };
    expect(flattenAlertRules(reorderedPayload)[0].id).toBe(rows[0].id);
  });

  it('maps recovering alert instances', () => {
    const payload: PrometheusAlertRulesResponse = {
      data: {
        groups: [
          {
            rules: [
              {
                name: 'recovering-rule',
                alerts: [
                  {
                    state: 'recovering',
                    labels: { alertname: 'Recovering alert' },
                    annotations: {},
                  },
                ],
              },
            ],
          },
        ],
      },
    };

    expect(flattenAlertRules(payload)[0].state).toBe('Recovering');
  });

  it('falls back to unknown-node when node_name label is absent', () => {
    const payload: PrometheusAlertRulesResponse = {
      data: {
        groups: [
          {
            rules: [
              {
                name: 'generic_alert',
                alerts: [
                  {
                    state: 'pending',
                    labels: {
                      alertname: 'Generic Alert',
                    },
                    annotations: {},
                  },
                ],
              },
            ],
          },
        ],
      },
    };

    const rows = flattenAlertRules(payload);

    expect(rows).toHaveLength(1);
    expect(rows[0].nodeId).toBe('');
    expect(rows[0].serviceName).toBe('');
  });

  it('flags rows as silenced when a suppressed Alertmanager alert matches', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-15T12:00:00.000Z'));

    const payload: PrometheusAlertRulesResponse = {
      data: {
        groups: [
          {
            rules: [
              {
                name: 'mysql_connections',
                alerts: [
                  {
                    state: 'firing',
                    labels: {
                      alertname: 'Silenced alert',
                      service_name: 'svc-a',
                    },
                    annotations: {},
                  },
                  {
                    state: 'firing',
                    labels: {
                      alertname: 'Loud alert',
                      service_name: 'svc-b',
                    },
                    annotations: {},
                  },
                ],
              },
            ],
          },
        ],
      },
    };

    // Extra private label on the Alertmanager side must not break correlation.
    const silenceMap = buildSilenceMap(
      [
        createSuppressedAlert(
          {
            alertname: 'Silenced alert',
            service_name: 'svc-a',
            __grafana_origin: 'grafana',
          },
          ['silence-1']
        ),
      ],
      [createSilence('silence-1', '2026-04-15T11:45:00.000Z')]
    );

    const rows = flattenAlertRules(payload, silenceMap);

    const silenced = rows.find((row) => row.alertName === 'Silenced alert');
    expect(silenced?.silenced).toBe(true);
    expect(silenced?.silencedBy).toEqual(['silence-1']);
    // Silenced 15 minutes ago (12:00 now, silence started 11:45).
    expect(silenced?.silencedAge).toBe('15m');

    const loud = rows.find((row) => row.alertName === 'Loud alert');
    expect(loud?.silenced).toBe(false);
    expect(loud?.silencedBy).toEqual([]);
    expect(loud?.silencedAge).toBe('-');
  });

  it('defaults silenced to false when no silence map is supplied', () => {
    const payload: PrometheusAlertRulesResponse = {
      data: {
        groups: [
          {
            rules: [
              {
                name: 'rule',
                alerts: [
                  {
                    state: 'firing',
                    labels: { alertname: 'Alert' },
                    annotations: {},
                  },
                ],
              },
            ],
          },
        ],
      },
    };

    expect(flattenAlertRules(payload)[0].silenced).toBe(false);
  });

  it('groups alert rows by node and exposes child alert rows', () => {
    const rows = [
      createAlertRow({
        type: 'alert' as const,
        id: 'a1',
        alertName: 'alert-1',
        ruleName: 'rule-1',
        state: 'Pending' as const,
        nodeId: 'node-a',
        serviceName: 'svc-a',
        summary: 's1',
        age: '1m',
      }),
      createAlertRow({
        type: 'alert' as const,
        id: 'a2',
        alertName: 'alert-2',
        ruleName: 'rule-2',
        state: 'Alerting' as const,
        nodeId: 'node-a',
        serviceName: 'svc-b',
        summary: 's2',
        age: '2m',
      }),
      createAlertRow({
        type: 'alert' as const,
        id: 'a3',
        alertName: 'alert-3',
        ruleName: 'rule-3',
        state: 'Normal' as const,
        nodeId: 'node-b',
        serviceName: 'svc-c',
        summary: 's3',
        age: '3m',
      }),
    ];

    const grouped = groupAlertsByNode(rows);

    expect(grouped).toHaveLength(2);
    expect(grouped[0].type).toBe('node');
    expect(grouped[0].nodeId).toBe('node-a');
    expect(grouped[0].alertCount).toBe(2);
    expect(grouped[0].state).toBe('Alerting');
    expect(grouped[0].alerts).toHaveLength(2);
    expect(grouped[0].alerts[0].type).toBe('alert');
  });
});

describe('alert links', () => {
  const createLinkedRow = (
    id: string,
    ruleGroupUid: string,
    labels: Record<string, string>
  ) =>
    createAlertRow({
      type: 'alert',
      id,
      alertName: id,
      ruleName: id,
      ruleGroupUid,
      state: 'Alerting',
      nodeId: '',
      serviceName: '',
      summary: '',
      age: '1m',
      labels,
    });

  const cpuOnNode1 = createLinkedRow('cpu-node-1', 'rule-cpu', {
    node_name: 'node-1',
    alertname: 'High CPU',
    __alert_rule_uid__: 'rule-cpu',
    __grafana_origin: 'plugin/percona',
  });
  const cpuOnNode2 = createLinkedRow('cpu-node-2', 'rule-cpu', {
    node_name: 'node-2',
    alertname: 'High CPU',
  });

  it('builds the link from the rule uid and the sorted visible labels', () => {
    expect(getAlertLinkParams(cpuOnNode1).toString()).toBe(
      'rule=rule-cpu&matcher=alertname%3DHigh+CPU&matcher=node_name%3Dnode-1'
    );
  });

  it('finds the alert a link points to', () => {
    const params = getAlertLinkParams(cpuOnNode2);

    expect(findLinkedAlert([cpuOnNode1, cpuOnNode2], params)).toBe(cpuOnNode2);
  });

  it('still finds the alert after a label is added to it', () => {
    const params = getAlertLinkParams(cpuOnNode1);
    const withNewLabel = createLinkedRow('cpu-node-1', 'rule-cpu', {
      ...cpuOnNode1.labels,
      environment: 'prod',
    });

    expect(findLinkedAlert([withNewLabel], params)).toBe(withNewLabel);
  });

  it('does not match an alert of another rule with the same labels', () => {
    const params = getAlertLinkParams(cpuOnNode1);
    const otherRule = createLinkedRow(
      'other',
      'rule-memory',
      cpuOnNode1.labels
    );

    expect(findLinkedAlert([otherRule], params)).toBeUndefined();
  });

  it('does not match when a linked label has a different value', () => {
    const params = getAlertLinkParams(cpuOnNode1);

    expect(findLinkedAlert([cpuOnNode2], params)).toBeUndefined();
  });

  it('round-trips label values with spaces, brackets, & and =', () => {
    const tricky = createLinkedRow('tricky', 'rule-cpu', {
      service_name: '[no value]',
      query: 'a=b&c=d',
    });
    const params = new URLSearchParams(getAlertLinkParams(tricky).toString());

    expect(findLinkedAlert([cpuOnNode1, tricky], params)).toBe(tricky);
  });

  it('sets the link, keeping other params and the tab last', () => {
    const params = new URLSearchParams(
      'kiosk=true&tab=raw-data&rule=old&matcher=node_name%3Dold'
    );

    expect(withAlertLink(params, cpuOnNode2).toString()).toBe(
      'kiosk=true&rule=rule-cpu&matcher=alertname%3DHigh+CPU&matcher=node_name%3Dnode-2&tab=raw-data'
    );
  });

  it('removes the link and tab without an alert', () => {
    const params = new URLSearchParams(
      'kiosk=true&rule=rule-cpu&matcher=node_name%3Dnode-1&tab=raw-data'
    );

    expect(withAlertLink(params).toString()).toBe('kiosk=true');
  });

  it('finds nothing without a rule param', () => {
    expect(
      findLinkedAlert([cpuOnNode1], new URLSearchParams('kiosk=true'))
    ).toBeUndefined();
  });
});
