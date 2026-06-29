import { PrometheusAlertRulesResponse } from 'types/alerting.types';
import {
  ALL_SERVICES_FILTER,
  ALL_NODES_FILTER,
  filterAlertRulesByNode,
  filterAlertRulesByService,
  flattenAlertRules,
  getServiceFilterOptionsForNode,
  groupAlertsByNode,
  getNodeFilterOptions,
  getServiceFilterOptions,
} from './AlertsPage.utils';
import { AlertRow } from './AlertsPage.types';

const createAlertRow = (
  row: Omit<AlertRow, 'labels' | 'annotations' | 'expression' | 'rawJson'> &
    Partial<Pick<AlertRow, 'labels' | 'annotations' | 'expression' | 'rawJson'>>
): AlertRow => ({
  labels: {},
  annotations: {},
  expression: '',
  rawJson: '{}',
  ...row,
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

  it('builds node/service options and filters by selected values', () => {
    const rows = [
      createAlertRow({
        type: 'alert' as const,
        id: 'r1',
        alertName: 'alert-1',
        ruleName: 'rule-1',
        state: 'Alerting' as const,
        nodeId: 'node-a',
        serviceName: 'svc-a',
        summary: 's1',
        source: 'src',
        age: '2m',
      }),
      createAlertRow({
        type: 'alert' as const,
        id: 'r2',
        alertName: 'alert-2',
        ruleName: 'rule-2',
        state: 'Pending' as const,
        nodeId: 'node-b',
        serviceName: 'svc-b',
        summary: 's2',
        source: 'src',
        age: '3m',
      }),
      createAlertRow({
        type: 'alert' as const,
        id: 'r3',
        alertName: 'alert-3',
        ruleName: 'rule-3',
        state: 'Pending' as const,
        nodeId: '',
        serviceName: '',
        summary: 's3',
        source: 'src',
        age: '4m',
      }),
    ];

    const options = getNodeFilterOptions(rows);
    expect(options.map((option) => option.value)).toEqual([
      ALL_NODES_FILTER,
      'node-a',
      'node-b',
    ]);

    const allNodeRows = filterAlertRulesByNode(rows, ALL_NODES_FILTER);
    expect(allNodeRows).toHaveLength(3);

    const nodeBRows = filterAlertRulesByNode(rows, 'node-b');
    expect(nodeBRows).toHaveLength(1);
    expect(nodeBRows[0].id).toBe('r2');

    const serviceOptions = getServiceFilterOptions(rows);
    expect(serviceOptions.map((option) => option.value)).toEqual([
      ALL_SERVICES_FILTER,
      'svc-a',
      'svc-b',
    ]);

    const allServicesRows = filterAlertRulesByService(
      rows,
      ALL_SERVICES_FILTER
    );
    expect(allServicesRows).toHaveLength(3);

    const serviceBRows = filterAlertRulesByService(rows, 'svc-b');
    expect(serviceBRows).toHaveLength(1);
    expect(serviceBRows[0].id).toBe('r2');

    const nodeBServiceOptions = getServiceFilterOptionsForNode(rows, 'node-b');
    expect(nodeBServiceOptions.map((option) => option.value)).toEqual([
      ALL_SERVICES_FILTER,
      'svc-b',
    ]);
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
        source: 'src',
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
        source: 'src',
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
        source: 'src',
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
