import { NAV_MYSQL, NAV_OS } from 'contexts/navigation/navigation.constants';
import type { ManagedService } from 'types/services.types';
import { mapNodes, mapServices, serviceTypeKey } from './inventory.source';

const service = (overrides: Partial<ManagedService>): ManagedService => ({
  serviceId: 'svc-1',
  serviceType: 'mysql',
  serviceName: 'mysql-prod 1',
  databaseName: '',
  nodeId: 'node-1',
  nodeName: 'db-host',
  environment: 'prod',
  cluster: 'c1',
  replicationSet: '',
  customLabels: {},
  externalGroup: '',
  address: '10.0.0.1',
  port: 3306,
  socket: '',
  version: '8.0',
  ...overrides,
});

describe('inventory.source', () => {
  it.each([
    ['mysql', 'mysql'],
    ['SERVICE_TYPE_POSTGRESQL_SERVICE', 'postgresql'],
    ['SERVICE_TYPE_EXTERNAL_SERVICE', 'external'],
  ])('reads service type %s as %s', (type, key) => {
    expect(serviceTypeKey(type)).toBe(key);
  });

  it('opens a service on its technology summary, filtered to it', () => {
    const [mysql] = mapServices([service({})]);

    expect(mysql).toMatchObject({
      id: 'service:svc-1',
      kind: 'service',
      group: 'inventory',
      label: 'mysql-prod 1',
      description: ['Inventory', 'MySQL on db-host'],
      navIcon: NAV_MYSQL.icon,
      to: '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=mysql-prod%201',
      remote: true,
      dedupeKey: 'service:svc-1',
    });
    expect(mysql.keywords).toEqual([
      'db-host',
      'prod',
      'c1',
      '10.0.0.1',
      'mysql',
    ]);
    expect(mysql.aliases).toEqual([
      'mariadb',
      'percona server',
      'pxc',
      'galera',
    ]);
  });

  it.each([
    [
      'mongodb',
      'MongoDB',
      '/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-service_name=s',
    ],
    [
      'SERVICE_TYPE_POSTGRESQL_SERVICE',
      'PostgreSQL',
      '/graph/d/postgresql-instance-summary/postgresql-instance-summary?var-service_name=s',
    ],
    [
      'valkey',
      'Valkey',
      '/graph/d/valkey-overview/valkey-redis-overview?var-service_name=s',
    ],
    [
      'proxysql',
      'ProxySQL',
      '/graph/d/proxysql-instance-summary?var-service_name=s',
    ],
    [
      'haproxy',
      'HAProxy',
      '/graph/d/haproxy-instance-summary?var-service_name=s',
    ],
    ['external', 'External', '/graph/inventory/services'],
  ])('maps a %s service to %s', (serviceType, name, to) => {
    const [item] = mapServices([service({ serviceType, serviceName: 's' })]);

    expect(item.description).toEqual(['Inventory', `${name} on db-host`]);
    expect(item.to).toBe(to);
  });

  it('keeps each service separate though they share a dashboard', () => {
    const items = mapServices([
      service({ serviceId: 'a' }),
      service({ serviceId: 'b' }),
    ]);

    expect(new Set(items.map((item) => item.dedupeKey)).size).toBe(2);
  });

  it('opens a node on its summary, filtered to it', () => {
    expect(
      mapNodes([
        {
          nodeId: 'n1',
          nodeName: 'pmm server',
          nodeType: 'generic',
          address: '127.0.0.1',
        },
      ])
    ).toEqual([
      expect.objectContaining({
        id: 'node:n1',
        kind: 'node',
        group: 'inventory',
        label: 'pmm server',
        description: ['Inventory', 'Node · 127.0.0.1'],
        navIcon: NAV_OS.icon,
        to: '/graph/d/node-instance-summary/node-summary?var-node_name=pmm%20server',
        remote: true,
      }),
    ]);
  });
});
