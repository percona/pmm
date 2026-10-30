import { describe, expect, it } from 'vitest';
import { AgentStatus, AgentType, type ManagedAgent } from 'types/agent.types';
import { ManagedServiceType } from 'types/services.types';
import { TEST_MANAGED_SERVICE } from 'utils/testStubs';
import { MonitoringStatus } from './Inventory.types';
import {
  getAgentsMonitoringStatus,
  getFilteredTabUrl,
  getInventoryRedirectUrl,
  getServiceDashboardUrl,
  getServiceQanUrl,
} from './Inventory.utils';

const agent = (
  agentType: AgentType,
  status: AgentStatus,
  disabled = false
): ManagedAgent => ({
  agentId: `${agentType}-${status}`,
  agentType,
  status,
  disabled,
  isConnected: false,
  properties: {},
});

describe('getAgentsMonitoringStatus', () => {
  it.each<[string, ManagedAgent[], MonitoringStatus]>([
    ['no agents', [], MonitoringStatus.ok],
    [
      'all agents running',
      [
        agent(AgentType.mysqldExporter, AgentStatus.running),
        agent(AgentType.nodeExporter, AgentStatus.starting),
      ],
      MonitoringStatus.ok,
    ],
    [
      'a failing agent',
      [
        agent(AgentType.mysqldExporter, AgentStatus.running),
        agent(AgentType.nodeExporter, AgentStatus.waiting),
      ],
      MonitoringStatus.failed,
    ],
    [
      'an agent that failed to start',
      [
        agent(AgentType.mysqldExporter, AgentStatus.running),
        agent(AgentType.qanMysqlSlowlogAgent, AgentStatus.initializationError),
      ],
      MonitoringStatus.failed,
    ],
    [
      'only pg_stat_statements disabled',
      [
        agent(AgentType.postgresExporter, AgentStatus.running),
        agent(AgentType.qanPostgresqlPgstatementsAgent, AgentStatus.done, true),
      ],
      MonitoringStatus.ok,
    ],
    [
      'another single agent disabled',
      [agent(AgentType.nodeExporter, AgentStatus.running, true)],
      MonitoringStatus.warning,
    ],
    [
      'several agents disabled',
      [
        agent(AgentType.mysqldExporter, AgentStatus.running),
        agent(AgentType.qanMysqlSlowlogAgent, AgentStatus.done, true),
        agent(AgentType.qanPostgresqlPgstatementsAgent, AgentStatus.done, true),
      ],
      MonitoringStatus.warning,
    ],
    [
      'only the RTA agent disabled',
      [agent(AgentType.rtaMongodbAgent, AgentStatus.done, true)],
      MonitoringStatus.ok,
    ],
  ])('with %s', (_, agents, status) => {
    expect(getAgentsMonitoringStatus(agents)).toBe(status);
  });

  it('counts a connected agent without status as running', () => {
    expect(
      getAgentsMonitoringStatus([
        {
          ...agent(AgentType.pmmAgent, AgentStatus.unknown),
          isConnected: true,
        },
      ])
    ).toBe(MonitoringStatus.ok);
  });
});

describe('getInventoryRedirectUrl', () => {
  it.each([
    ['', '', '/inventory/services'],
    ['services', '', '/inventory/services'],
    [
      'services',
      '?search-text-input=service-1&search-select=serviceId',
      '/inventory/services?services.f.serviceId=service-1',
    ],
    [
      'nodes',
      '?search-text-input=node-1&search-select=nodeId',
      '/inventory/nodes?nodes.f.nodeId=node-1',
    ],
    [
      'nodes',
      '?isPmmServerNode=true',
      '/inventory/nodes?nodes.f.isPmmServerNode=true',
    ],
    [
      'services',
      '?search-text-input=mysql&search-select=All&status=N%2FA&agentsStatus=Failed&orgId=1',
      '/inventory/services?services.f.status=STATUS_UNSPECIFIED&services.f.agentsStatus=Failed&services.q=mysql',
    ],
    [
      'services/service-1/agents',
      '?type=node_exporter&status=AGENT_STATUS_RUNNING',
      '/inventory/services/service-1/agents?agents.f.type=node_exporter&agents.f.status=AGENT_STATUS_RUNNING',
    ],
    [
      'nodes',
      '?nodes.f.nodeName=node-1',
      '/inventory/nodes?nodes.f.nodeName=node-1',
    ],
  ])('maps %s%s', (path, search, url) => {
    expect(getInventoryRedirectUrl(path, search)).toBe(url);
  });
});

describe('links', () => {
  it('filters a tab on a column', () => {
    expect(getFilteredTabUrl('nodes', 'nodeId', 'node-1')).toBe(
      '/inventory/nodes?nodes.f.nodeId=node-1'
    );
  });

  it('opens the service dashboard and QAN for the service', () => {
    const service = { ...TEST_MANAGED_SERVICE, serviceName: 'my service' };

    expect(getServiceDashboardUrl(service)).toBe(
      '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=my%20service'
    );
    expect(getServiceQanUrl(service)).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?var-service_name=my%20service'
    );
  });

  it('opens the Valkey overview, which has no QAN', () => {
    const service = {
      ...TEST_MANAGED_SERVICE,
      serviceType: ManagedServiceType.valkey,
      serviceName: 'valkey-1',
    };

    expect(getServiceDashboardUrl(service)).toBe(
      '/graph/d/valkey-overview/valkey-redis-overview?var-service_name=valkey-1'
    );
    expect(getServiceQanUrl(service)).toBeUndefined();
  });

  it('links nowhere for an external service', () => {
    const service = {
      ...TEST_MANAGED_SERVICE,
      serviceType: ManagedServiceType.external,
    };

    expect(getServiceDashboardUrl(service)).toBeUndefined();
    expect(getServiceQanUrl(service)).toBeUndefined();
  });
});
