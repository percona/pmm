import { describe, expect, it, vi } from 'vitest';
import { AgentStatus, AgentType, type ManagedAgent } from 'types/agent.types';
import { ServiceStatus } from 'types/services.types';
import { TEST_MANAGED_SERVICE } from 'utils/testStubs';
import { toServiceRows } from '../Services.utils';
import { exportServicesToCsv, mapServiceToCsvRow } from './exportServicesToCsv';

const { download, generateCsv, mkConfig } = vi.hoisted(() => ({
  download: vi.fn(() => vi.fn()),
  generateCsv: vi.fn(() => vi.fn(() => 'csv-content')),
  mkConfig: vi.fn((config) => config),
}));

vi.mock('export-to-csv', () => ({
  download,
  generateCsv,
  mkConfig,
}));

const agent = (
  agentId: string,
  agentType: AgentType,
  status: AgentStatus,
  disabled = false
): ManagedAgent => ({
  agentId,
  agentType,
  status,
  isConnected: false,
  disabled,
  properties: {},
});

const [SERVICE] = toServiceRows([
  {
    ...TEST_MANAGED_SERVICE,
    serviceName: '=mysql-1',
    status: ServiceStatus.down,
    agents: [
      agent('exporter', AgentType.mysqldExporter, AgentStatus.running),
      agent(
        'perfschema',
        AgentType.qanMysqlPerfschemaAgent,
        AgentStatus.initializationError
      ),
      agent('slowlog', AgentType.qanMysqlSlowlogAgent, AgentStatus.waiting),
      agent('disabled', AgentType.nodeExporter, AgentStatus.done, true),
    ],
    labels: {
      zone: 'eu-1',
      environment: 'production',
      cluster: 'cluster-1',
      version: '8.0.0',
    },
  },
]);

describe('exportServicesToCsv', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('maps a service as the page shows it, naming its failed agents', () => {
    expect(mapServiceToCsvRow(SERVICE)).toEqual({
      service_id: 'service-1',
      service_name: "'=mysql-1",
      service_type: 'mysql',
      status: 'Down',
      monitoring: 'Failed',
      failed_agents:
        'QAN MySQL Performance Schema agent (perfschema): Initialization error; ' +
        'QAN MySQL Slow Query Log agent (slowlog): Waiting',
      node_id: 'node-1',
      node_name: 'Node 1',
      address: '127.0.0.1',
      port: 3306,
      zone: 'eu-1',
      environment: 'production',
      cluster: 'cluster-1',
      version: '8.0.0',
    });
  });

  it('leaves the failed agents empty while monitoring is fine', () => {
    const [service] = toServiceRows([
      {
        ...TEST_MANAGED_SERVICE,
        agents: [
          agent('exporter', AgentType.mysqldExporter, AgentStatus.running),
        ],
      },
    ]);

    expect(mapServiceToCsvRow(service)).toMatchObject({
      monitoring: 'OK',
      failed_agents: '',
    });
  });

  it('keeps custom labels named like an exported field under their own header', () => {
    const [service] = toServiceRows([
      {
        ...TEST_MANAGED_SERVICE,
        agents: [
          agent('exporter', AgentType.mysqldExporter, AgentStatus.running),
        ],
        labels: {
          monitoring: 'payments',
          failed_agents: 'none',
          status: 'custom',
          team: 'payments',
        },
      },
    ]);

    expect(mapServiceToCsvRow(service)).toMatchObject({
      monitoring: 'OK',
      failed_agents: '',
      status: 'Up',
      'label:monitoring': 'payments',
      'label:failed_agents': 'none',
      'label:status': 'custom',
      team: 'payments',
    });
  });

  it('puts the table columns first and the other labels alphabetically, whatever the service order', () => {
    const [other] = toServiceRows([
      {
        ...TEST_MANAGED_SERVICE,
        serviceId: 'service-2',
        labels: { team: 'payments', az: 'a' },
      },
    ]);

    exportServicesToCsv([SERVICE, other]);
    exportServicesToCsv([other, SERVICE]);

    const expected = [
      'service_id',
      'service_name',
      'service_type',
      'status',
      'monitoring',
      'failed_agents',
      'node_id',
      'node_name',
      'address',
      'port',
      'environment',
      'cluster',
      'az',
      'team',
      'version',
      'zone',
    ];
    expect(mkConfig).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ columnHeaders: expected })
    );
    expect(mkConfig).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ columnHeaders: expected })
    );
  });

  it('downloads the services as csv', () => {
    exportServicesToCsv([SERVICE]);

    expect(mkConfig).toHaveBeenCalledWith({
      columnHeaders: expect.arrayContaining(['service_id', 'failed_agents']),
      filename: expect.stringMatching(
        /^inventory_services_export_\d{8}_\d{6}$/
      ),
    });
    expect(download).toHaveBeenCalled();
  });
});
