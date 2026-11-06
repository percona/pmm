import type { RdsInstanceRow } from 'types/rds.types';
import {
  ALERT_RULES,
  GIB,
  RDS_ROWS,
  rdsRow,
  SILENCED_ALERTS,
} from './AmazonRds.mocks';
import { NO_FILTERS } from './AmazonRds.constants';
import type {
  GroupFilters,
  InstanceGroup,
  RdsInstance,
} from './AmazonRds.types';
import {
  addAlerts,
  compareSortValues,
  filterGroups,
  formatBytes,
  formatStorage,
  formatThroughput,
  getAccountOptions,
  getEngine,
  getAlertGroupsUrl,
  getAlertRuleUrl,
  getAlertsBand,
  getEngineOptions,
  getFiringAlerts,
  getGroupOptions,
  getGroupType,
  getInstanceDetails,
  getInstanceLinks,
  getRegionOptions,
  getRole,
  getTypeOptions,
  getWriters,
  groupInstances,
  narrowSelection,
  readFilters,
  sortGroups,
  writeFilters,
} from './AmazonRds.utils';

const groups = groupInstances(RDS_ROWS);

const findGroup = (name: string, engine?: string) =>
  groups.find(
    (group) => group.name === name && (!engine || group.engine === engine)
  ) as InstanceGroup;

const row = (values: Partial<RdsInstanceRow>): RdsInstanceRow =>
  rdsRow(values.serviceName ?? 'svc', {
    awsAccountId: '111111111111',
    awsEngine: 'mysql',
    instanceId: values.serviceName ?? 'svc',
    ...values,
  });

const instance = (values: Partial<RdsInstance>): RdsInstance => ({
  serviceName: 'db-1',
  nodeName: 'db-1',
  instanceId: 'db-1',
  region: 'eu-north-1',
  role: 'writer',
  ...values,
});

describe('AmazonRds.utils: grouping', () => {
  it('groups clusters, replication groups and standalone instances, without AWS details last in a region', () => {
    expect(
      groups.map(({ name, kind, engine }) => [name, kind, engine])
    ).toEqual([
      ['database-2', 'cluster', 'aurora-mysql'],
      ['database-3', 'cluster', 'aurora-mysql'],
      ['database-4', 'cluster', 'aurora-postgresql'],
      ['AWS info unavailable', 'unknown', 'mysql'],
      ['AWS info unavailable', 'unknown', 'postgres'],
      ['rds-mariadb-1', 'instance', 'mariadb'],
      ['rds-mysql-1', 'replication', 'mysql'],
      ['rds-pg-1', 'instance', 'postgres'],
    ]);
  });

  it('aggregates an Aurora MySQL cluster: counts, writers, summed queries and max volume', () => {
    expect(findGroup('database-3')).toMatchObject({
      region: 'eu-north-1',
      awsAccountId: '111111111111',
      monitoredInstances: 2,
      exportersConnected: 2,
      writers: 1,
      unknownRoles: 0,
      throughput: 24.75,
      storage: { kind: 'used', bytes: 5000 },
    });
    expect(
      findGroup('database-3').instances.map(({ serviceName, role }) => [
        serviceName,
        role,
      ])
    ).toEqual([
      ['database-3-instance-1', 'writer'],
      ['database-3-instance-1-reader', 'reader'],
    ]);
  });

  it('takes the volume of an Aurora PostgreSQL cluster from any instance that reports it', () => {
    const database4 = findGroup('database-4');

    expect(database4).toMatchObject({
      writers: 1,
      throughput: 17,
      storage: { kind: 'used', bytes: 3000 },
    });
    expect(
      database4.instances.map(({ role, storage }) => [role, storage])
    ).toEqual([
      ['writer', undefined],
      ['reader', undefined],
    ]);
  });

  it('groups a primary with its read replicas, including one in another region', () => {
    const replication = findGroup('rds-mysql-1');

    expect(replication).toMatchObject({
      region: 'us-east-2',
      monitoredInstances: 3,
      writers: 1,
      throughput: 36,
      storage: { kind: 'free', bytes: 18 * GIB, lowest: true },
    });
    expect(
      replication.instances.map((item) => [
        item.serviceName,
        item.role,
        item.region,
      ])
    ).toEqual([
      ['rds-mysql-1', 'writer', 'us-east-2'],
      ['rds-mysql-1-replica', 'reader', 'us-east-2'],
      ['rds-mysql-1-west', 'reader', 'us-west-2'],
    ]);
  });

  it('keeps a standalone instance as its own group with its free storage', () => {
    expect(findGroup('rds-pg-1')).toMatchObject({
      monitoredInstances: 1,
      writers: 1,
      storage: { kind: 'free', bytes: 50 * GIB },
    });
    expect(findGroup('rds-pg-1').instances[0].storage).toEqual({
      kind: 'free',
      bytes: 50 * GIB,
    });
  });

  it('puts instances without AWS details in one group per region and engine', () => {
    const mysql = findGroup('AWS info unavailable', 'mysql');

    expect(mysql).toMatchObject({
      awsAccountId: '',
      exportersConnected: 0,
      writers: 0,
      unknownRoles: 1,
    });
    expect(mysql.instances).toEqual([
      {
        serviceName: 'aurora-stale',
        nodeName: '',
        instanceId: '',
        region: 'eu-north-1',
        engine: 'mysql',
        role: 'unknown',
        up: 0,
        throughput: undefined,
        storage: undefined,
      },
    ]);
    expect(
      findGroup('AWS info unavailable', 'postgres').instances.map(
        ({ serviceName }) => serviceName
      )
    ).toEqual(['legacy-pg']);
  });

  it('joins cascaded replicas to the root primary', () => {
    const [group] = groupInstances([
      row({ serviceName: 'primary' }),
      row({
        serviceName: 'replica-1',
        sourceInstanceId: 'primary',
        sourceRegion: 'eu-north-1',
      }),
      row({
        serviceName: 'replica-2',
        sourceInstanceId: 'replica-1',
        sourceRegion: 'eu-north-1',
      }),
    ]);

    expect(group).toMatchObject({
      kind: 'replication',
      name: 'primary',
      monitoredInstances: 3,
    });
  });

  it('names a replication group after a primary PMM does not monitor', () => {
    const replicaGroups = groupInstances([
      row({
        serviceName: 'replica-1',
        sourceInstanceId: 'primary',
        sourceRegion: 'eu-north-1',
      }),
      row({ serviceName: 'replica-2', sourceInstanceId: 'primary' }),
    ]);

    expect(replicaGroups).toHaveLength(1);
    expect(replicaGroups[0]).toMatchObject({
      kind: 'replication',
      name: 'primary',
      monitoredInstances: 2,
    });
  });

  it('groups a Multi-AZ DB cluster by cluster ID with its lowest free storage', () => {
    const [group] = groupInstances([
      row({
        serviceName: 'multi-az-1',
        clusterId: 'multi-az',
        freeStorageBytes: 30,
      }),
      row({
        serviceName: 'multi-az-2',
        clusterId: 'multi-az',
        freeStorageBytes: 20,
      }),
    ]);

    expect(group).toMatchObject({
      kind: 'cluster',
      name: 'multi-az',
      storage: { kind: 'free', bytes: 20, lowest: true },
    });
    expect(getGroupType(group)).toBe('multi-az-cluster');
  });
});

describe('AmazonRds.utils: roles and engines', () => {
  it('reads the engine from aws_engine, falling back to the service type', () => {
    expect(getEngine(row({ awsEngine: 'aurora-postgresql' }))).toBe(
      'aurora-postgresql'
    );
    expect(getEngine(row({ awsEngine: 'aurora' }))).toBe('aurora-mysql');
    expect(getEngine(row({ awsEngine: '', serviceType: 'postgresql' }))).toBe(
      'postgres'
    );
    expect(getEngine(row({ awsEngine: '', serviceType: 'mysql' }))).toBe(
      'mysql'
    );
    expect(getEngine(row({ awsEngine: '', serviceType: '' }))).toBeUndefined();
    expect(
      getEngine(row({ awsEngine: 'constructor', serviceType: '' }))
    ).toBeUndefined();
  });

  it('takes the Aurora MySQL role from innodb_read_only', () => {
    const aurora = { awsEngine: 'aurora-mysql', readOnly: 0 };

    expect(getRole(row({ ...aurora, innodbReadOnly: 0 }), 'aurora-mysql')).toBe(
      'writer'
    );
    expect(getRole(row({ ...aurora, innodbReadOnly: 1 }), 'aurora-mysql')).toBe(
      'reader'
    );
    expect(getRole(row(aurora), 'aurora-mysql')).toBe('unknown');
  });

  it('takes the RDS MySQL and MariaDB role from read_only', () => {
    expect(getRole(row({ innodbReadOnly: 0, readOnly: 1 }), 'mysql')).toBe(
      'reader'
    );
    expect(getRole(row({ innodbReadOnly: 0, readOnly: 0 }), 'mysql')).toBe(
      'writer'
    );
    expect(getRole(row({ awsEngine: 'mariadb', readOnly: 1 }), 'mariadb')).toBe(
      'reader'
    );
    expect(getRole(row({ innodbReadOnly: 0 }), 'mysql')).toBe('unknown');
  });

  it('takes the PostgreSQL role from recovery, for Aurora and RDS', () => {
    expect(
      getRole(row({ awsEngine: 'postgres', inRecovery: 0 }), 'postgres')
    ).toBe('writer');
    expect(
      getRole(
        row({ awsEngine: 'aurora-postgresql', inRecovery: 1 }),
        'aurora-postgresql'
      )
    ).toBe('reader');
    expect(
      getRole(
        row({ awsEngine: 'aurora-postgresql', readOnly: 0 }),
        'aurora-postgresql'
      )
    ).toBe('unknown');
  });

  it('marks a MySQL reader by either variable without AWS details', () => {
    const noInfo = { awsEngine: '' };

    expect(
      getRole(row({ ...noInfo, innodbReadOnly: 1, readOnly: 0 }), 'mysql')
    ).toBe('reader');
    expect(
      getRole(row({ ...noInfo, innodbReadOnly: 0, readOnly: 1 }), 'mysql')
    ).toBe('reader');
    expect(
      getRole(row({ ...noInfo, innodbReadOnly: 0, readOnly: 0 }), 'mysql')
    ).toBe('writer');
    expect(getRole(row(noInfo), 'mysql')).toBe('unknown');
  });

  it('describes the writers of a group', () => {
    const group = (writers: number, unknownRoles = 0) =>
      ({ writers, unknownRoles }) as InstanceGroup;

    expect(getWriters(group(1, 1))).toEqual({
      text: '1 writer',
      state: 'good',
    });
    expect(getWriters(group(2))).toEqual({
      text: '2 writers',
      state: 'warning',
    });
    expect(getWriters(group(0))).toEqual({
      text: 'no writer',
      state: 'warning',
    });
    expect(getWriters(group(0, 2))).toEqual({
      text: 'unknown',
      state: 'unknown',
    });
  });
});

describe('AmazonRds.utils: formatting and links', () => {
  it('formats bytes with binary units', () => {
    expect(formatBytes(512)).toBe('512 B');
    expect(formatBytes(765 * 1024 * 1024)).toBe('765 MiB');
    expect(formatBytes(1.5 * GIB)).toBe('1.5 GiB');
    expect(formatBytes(2048 * GIB)).toBe('2 TiB');
  });

  it('formats throughput as queries/s for MySQL and transactions/s for PostgreSQL', () => {
    expect(formatThroughput(24.75, 'aurora-mysql')).toBe('24.75 qps');
    expect(formatThroughput(3, 'mariadb')).toBe('3.00 qps');
    expect(formatThroughput(17, 'aurora-postgresql')).toBe('17.00 tps');
    expect(formatThroughput(12345.678, 'mysql')).toBe('12,345.68 qps');
    expect(formatThroughput(2, undefined)).toBe('2.00');
    expect(formatThroughput(undefined, 'postgres')).toBe('-');
  });

  it('labels Aurora volume as used and other storage as free', () => {
    expect(formatStorage({ kind: 'used', bytes: 765 * 1024 * 1024 })).toBe(
      '765 MiB used'
    );
    expect(formatStorage({ kind: 'free', bytes: 18 * GIB })).toBe(
      '18 GiB free'
    );
    expect(formatStorage({ kind: 'free', bytes: 18 * GIB, lowest: true })).toBe(
      '18 GiB free (min)'
    );
    expect(formatStorage(undefined)).toBe('-');
  });

  it('links an Aurora MySQL instance to its details, with MySQL and OS links', () => {
    expect(getInstanceLinks(instance({ engine: 'aurora-mysql' }))).toEqual({
      service: {
        label: 'db-1',
        title: 'MySQL Amazon Aurora Details for db-1',
        url: '/graph/d/mysql-amazonaurora/mysql-amazon-aurora-details?var-service_name=db-1',
      },
      dashboards: [
        {
          label: 'MySQL',
          title: 'MySQL Instance Summary for db-1',
          url: '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=db-1',
        },
        {
          label: 'OS',
          title: 'Node Summary for db-1',
          url: '/graph/d/node-instance-summary/node-summary?var-node_name=db-1',
        },
      ],
    });
  });

  it.each([['mysql' as const], ['mariadb' as const], [undefined]])(
    'links a %s instance to its MySQL Instance Summary',
    (engine) => {
      const { service, dashboards } = getInstanceLinks(
        instance({ engine, serviceName: 'db 1' })
      );

      expect(service).toEqual({
        label: 'db 1',
        title: 'MySQL Instance Summary for db 1',
        url: '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=db%201',
      });
      expect(dashboards.map(({ label }) => label)).toEqual(['OS']);
    }
  );

  it.each([['aurora-postgresql' as const], ['postgres' as const]])(
    'links a %s instance to its PostgreSQL Instance Summary',
    (engine) => {
      const { service, dashboards } = getInstanceLinks(instance({ engine }));

      expect(service).toEqual({
        label: 'db-1',
        title: 'PostgreSQL Instance Summary for db-1',
        url: '/graph/d/postgresql-instance-summary/postgresql-instance-summary?var-service_name=db-1',
      });
      expect(dashboards.map(({ label }) => label)).toEqual(['OS']);
    }
  );

  it('links a node to its Node Summary, falling back to the service name', () => {
    const [os] = getInstanceLinks(
      instance({ engine: 'postgres', nodeName: '' })
    ).dashboards;

    expect(os.url).toBe(
      '/graph/d/node-instance-summary/node-summary?var-node_name=db-1'
    );
  });

  it('shows the instance ID and region only where they differ from the service and group', () => {
    const replication = findGroup('rds-mysql-1');
    const [primary, , west] = replication.instances;

    expect(getInstanceDetails(primary, replication)).toBe('');
    expect(getInstanceDetails(west, replication)).toBe(
      'rds-mysql-1-replica-west, us-west-2'
    );
    expect(
      getInstanceDetails(
        findGroup('AWS info unavailable').instances[0],
        replication
      )
    ).toBe('eu-north-1');
  });
});

describe('AmazonRds.utils: filters', () => {
  const values = (options: Array<{ value: string }>) =>
    options.map(({ value }) => value);
  const filter = (filters: Partial<GroupFilters>) =>
    filterGroups(groups, { ...NO_FILTERS, ...filters });
  const names = (filters: Partial<GroupFilters>) =>
    filter(filters).groups.map(({ name }) => name);

  it('keeps every group without filters', () => {
    expect(filter({})).toEqual({ groups, autoExpanded: new Set() });
  });

  it('searches group name, account, region and engine case-insensitively, keeping all instances', () => {
    expect(names({ query: 'DATABASE-3' })).toEqual(['database-3']);
    expect(names({ query: '2222' })).toEqual([
      'rds-mariadb-1',
      'rds-mysql-1',
      'rds-pg-1',
    ]);
    expect(names({ query: 'mariadb' })).toEqual(['rds-mariadb-1']);
    expect(names({ query: 'aurora postgresql' })).toEqual(['database-4']);
    expect(names({ query: 'us-west' })).toEqual([]);
    expect(filter({ query: 'rds-mysql-1' }).groups[0].instances).toHaveLength(
      3
    );
    expect(filter({ query: 'rds-mysql-1' }).autoExpanded.size).toBe(0);
  });

  it('narrows a group matched only by an instance to those instances and expands it', () => {
    const { groups: matching, autoExpanded } = filter({ query: ' Reader ' });

    expect(matching.map(({ name }) => name)).toEqual([
      'database-3',
      'database-4',
    ]);
    expect(matching[0]).toMatchObject({ monitoredInstances: 2 });
    expect(matching[0].instances.map(({ serviceName }) => serviceName)).toEqual(
      ['database-3-instance-1-reader']
    );
    expect(autoExpanded).toEqual(new Set(matching.map(({ key }) => key)));
  });

  it('finds an instance by its AWS instance ID', () => {
    expect(
      filter({ query: 'replica-west' }).groups[0].instances.map(
        ({ serviceName }) => serviceName
      )
    ).toEqual(['rds-mysql-1-west']);
  });

  it('filters by engine, alone and combined with the other filters', () => {
    expect(names({ engines: ['aurora-postgresql'] })).toEqual(['database-4']);
    expect(names({ engines: ['postgres'] })).toEqual([
      'AWS info unavailable',
      'rds-pg-1',
    ]);
    expect(
      names({ engines: ['mysql', 'mariadb'], regions: ['us-east-2'] })
    ).toEqual(['rds-mariadb-1', 'rds-mysql-1']);
    expect(
      names({ engines: ['aurora-mysql'], accounts: ['222222222222'] })
    ).toEqual([]);
  });

  it('combines account and group filters with the search', () => {
    expect(names({ accounts: ['111111111111'] })).toEqual([
      'database-2',
      'database-3',
      'database-4',
    ]);
    expect(
      names({ accounts: ['111111111111'], groups: ['database-3'] })
    ).toEqual(['database-3']);
    expect(
      names({ accounts: ['222222222222'], groups: ['database-3'] })
    ).toEqual([]);
    expect(
      names({
        accounts: ['111111111111', '222222222222'],
        query: 'instance-1',
      })
    ).toEqual(['database-2', 'database-3', 'database-4']);
  });

  it('derives the type of every kind of group', () => {
    expect(groups.map((group) => [group.name, getGroupType(group)])).toEqual([
      ['database-2', 'aurora-cluster'],
      ['database-3', 'aurora-cluster'],
      ['database-4', 'aurora-cluster'],
      ['AWS info unavailable', 'unavailable'],
      ['AWS info unavailable', 'unavailable'],
      ['rds-mariadb-1', 'instance'],
      ['rds-mysql-1', 'replication'],
      ['rds-pg-1', 'instance'],
    ]);
    expect(
      getGroupType({ kind: 'cluster', engine: 'postgres' } as InstanceGroup)
    ).toBe('multi-az-cluster');
    expect(
      getGroupType({
        kind: 'cluster',
        engine: 'aurora-postgresql',
      } as InstanceGroup)
    ).toBe('aurora-cluster');
  });

  it('filters by type and searches the type text', () => {
    expect(names({ types: ['aurora-cluster'] })).toEqual([
      'database-2',
      'database-3',
      'database-4',
    ]);
    expect(names({ types: ['instance', 'replication'] })).toEqual([
      'rds-mariadb-1',
      'rds-mysql-1',
      'rds-pg-1',
    ]);
    expect(names({ types: ['unavailable'] })).toEqual([
      'AWS info unavailable',
      'AWS info unavailable',
    ]);
    expect(
      names({ types: ['aurora-cluster'], engines: ['aurora-postgresql'] })
    ).toEqual(['database-4']);
    expect(names({ query: 'aurora cluster' })).toEqual([
      'database-2',
      'database-3',
      'database-4',
    ]);
    expect(names({ query: 'replication group' })).toEqual(['rds-mysql-1']);
  });

  it('builds type options from the data and narrows the DB identifier options by type', () => {
    expect(getTypeOptions(groups)).toEqual([
      { label: 'Aurora cluster', value: 'aurora-cluster' },
      { label: 'Instance', value: 'instance' },
      { label: 'Replication group', value: 'replication' },
      { label: 'Unavailable', value: 'unavailable' },
    ]);
    expect(
      values(
        getGroupOptions(groups, {
          regions: [],
          accounts: [],
          engines: [],
          types: ['instance', 'replication'],
        })
      )
    ).toEqual(['rds-mariadb-1', 'rds-mysql-1', 'rds-pg-1']);
    expect(
      narrowSelection(groups, {
        regions: [],
        accounts: [],
        engines: [],
        types: ['instance'],
        groups: ['database-3', 'rds-pg-1'],
      }).groups
    ).toEqual(['rds-pg-1']);
  });

  it('finds the groups without AWS details by search and by group filter', () => {
    expect(names({ query: 'unavailable' })).toEqual([
      'AWS info unavailable',
      'AWS info unavailable',
    ]);
    expect(names({ groups: ['AWS info unavailable'] })).toEqual([
      'AWS info unavailable',
      'AWS info unavailable',
    ]);
    expect(names({ query: 'stale' })).toEqual(['AWS info unavailable']);
  });

  it('returns nothing when no group matches', () => {
    expect(filter({ query: 'no-such-group' })).toEqual({
      groups: [],
      autoExpanded: new Set(),
    });
  });

  it('filters by region, alone and combined with the other filters', () => {
    expect(names({ regions: ['us-east-2'] })).toEqual([
      'rds-mariadb-1',
      'rds-mysql-1',
      'rds-pg-1',
    ]);
    expect(names({ regions: ['us-east-2'], groups: ['database-3'] })).toEqual(
      []
    );
    expect(names({ regions: ['eu-north-1'], query: 'reader' })).toEqual([
      'database-3',
      'database-4',
    ]);
  });

  it('builds region, account and engine options from the data', () => {
    expect(getRegionOptions(groups)).toEqual([
      { label: 'eu-north-1', value: 'eu-north-1' },
      { label: 'us-east-2', value: 'us-east-2' },
    ]);
    expect(getAccountOptions(groups, [])).toEqual([
      {
        label: '111111111111',
        value: '111111111111',
        description: 'eu-north-1',
      },
      {
        label: '222222222222',
        value: '222222222222',
        description: 'us-east-2',
      },
    ]);
    expect(values(getAccountOptions(groups, ['us-east-2']))).toEqual([
      '222222222222',
    ]);
    expect(getEngineOptions(groups)).toEqual([
      { label: 'Aurora MySQL', value: 'aurora-mysql' },
      { label: 'Aurora PostgreSQL', value: 'aurora-postgresql' },
      { label: 'MariaDB', value: 'mariadb' },
      { label: 'MySQL', value: 'mysql' },
      { label: 'PostgreSQL', value: 'postgres' },
    ]);
  });

  it('builds group options narrowed to the selected regions, accounts and engines, unknown group last', () => {
    const selection = { regions: [], accounts: [], engines: [], types: [] };

    expect(values(getGroupOptions(groups, selection))).toEqual([
      'database-2',
      'database-3',
      'database-4',
      'rds-mariadb-1',
      'rds-mysql-1',
      'rds-pg-1',
      'AWS info unavailable',
    ]);
    expect(
      values(
        getGroupOptions(groups, { ...selection, accounts: ['222222222222'] })
      )
    ).toEqual(['rds-mariadb-1', 'rds-mysql-1', 'rds-pg-1']);
    expect(
      values(
        getGroupOptions(groups, {
          ...selection,
          engines: ['postgres', 'aurora-postgresql'],
        })
      )
    ).toEqual(['database-4', 'rds-pg-1', 'AWS info unavailable']);
  });

  it('drops selected accounts and groups the selected regions, accounts and engines no longer offer', () => {
    const selection = {
      regions: [],
      accounts: ['111111111111', '222222222222'],
      engines: [],
      types: [],
      groups: ['database-3', 'rds-mysql-1'],
    };

    expect(
      narrowSelection(groups, { ...selection, regions: ['us-east-2'] })
    ).toEqual({
      ...selection,
      regions: ['us-east-2'],
      accounts: ['222222222222'],
      groups: ['rds-mysql-1'],
    });
    expect(
      narrowSelection(groups, { ...selection, engines: ['aurora-mysql'] })
    ).toEqual({
      ...selection,
      engines: ['aurora-mysql'],
      groups: ['database-3'],
    });
    expect(narrowSelection(groups, selection)).toEqual(selection);
  });
});

describe('AmazonRds.utils: URL', () => {
  it('reads filters from URL parameters', () => {
    expect(readFilters(new URLSearchParams())).toEqual(NO_FILTERS);
    expect(
      readFilters(
        new URLSearchParams([
          ['rds-q', 'reader'],
          ['rds-region', 'eu-north-1'],
          ['rds-region', 'us-east-2'],
          ['rds-account', '111111111111'],
          ['rds-engine', 'postgres'],
          ['rds-type', 'aurora-cluster'],
          ['rds-id', 'database-2'],
          ['rds-id', 'database-3'],
          ['rds-id', ''],
        ])
      )
    ).toEqual({
      query: 'reader',
      regions: ['eu-north-1', 'us-east-2'],
      accounts: ['111111111111'],
      engines: ['postgres'],
      types: ['aurora-cluster'],
      groups: ['database-2', 'database-3'],
    });
  });

  it('writes filters to URL parameters, dropping empty ones and keeping others', () => {
    const params = new URLSearchParams(
      'theme=dark&rds-q=old&rds-region=eu-north-1'
    );

    expect(writeFilters(NO_FILTERS, params).toString()).toBe('theme=dark');
    expect(
      writeFilters(
        {
          query: 'db 1',
          regions: ['r'],
          accounts: ['1'],
          engines: ['mysql', 'postgres'],
          types: ['instance'],
          groups: ['g'],
        },
        params
      ).toString()
    ).toBe(
      'theme=dark&rds-q=db+1&rds-region=r&rds-account=1&rds-engine=mysql&rds-engine=postgres&rds-type=instance&rds-id=g'
    );
    expect(params.toString()).toBe(
      'theme=dark&rds-q=old&rds-region=eu-north-1'
    );
  });

  it('reads back the filters it writes', () => {
    const filters: GroupFilters = {
      ...NO_FILTERS,
      query: 'a&b=c',
      groups: ['x y'],
    };

    expect(readFilters(writeFilters(filters, new URLSearchParams()))).toEqual(
      filters
    );
  });
});

describe('AmazonRds.utils: sorting', () => {
  const sortedNames = (id: string, desc: boolean) =>
    sortGroups(groups, [{ id, desc }]).map(({ name }) => name);

  it('sorts missing values last in both directions', () => {
    expect(compareSortValues(undefined, 1, false)).toBe(1);
    expect(compareSortValues(undefined, 1, true)).toBe(1);
    expect(compareSortValues(1, undefined, true)).toBe(-1);
    expect(compareSortValues(1, 3, false)).toBeLessThan(0);
    expect(compareSortValues(1, 3, true)).toBeGreaterThan(0);
    expect(compareSortValues('b', 'a', false)).toBeGreaterThan(0);
  });

  it('sorts groups by a column, keeping groups without a value last', () => {
    expect(sortedNames('throughput', true)).toEqual([
      'rds-mysql-1',
      'database-3',
      'database-4',
      'database-2',
      'rds-pg-1',
      'rds-mariadb-1',
      'AWS info unavailable',
      'AWS info unavailable',
    ]);
    expect(sortedNames('storage', false).slice(-2)).toEqual([
      'AWS info unavailable',
      'AWS info unavailable',
    ]);
    expect(sortedNames('type', false)[0]).toBe('database-2');
    expect(sortedNames('type', true)[0]).toBe('AWS info unavailable');
  });

  it('keeps the order without sorting or for an unknown column', () => {
    expect(sortGroups(groups, [])).toBe(groups);
    expect(sortGroups(groups, [{ id: 'mrt-row-expand', desc: false }])).toBe(
      groups
    );
  });
});

describe('AmazonRds.utils: alerts', () => {
  const alerts = getFiringAlerts(ALERT_RULES, SILENCED_ALERTS);
  const withAlerts = addAlerts(groups, alerts);
  const alertsOf = (name: string) =>
    withAlerts.find((group) => group.name === name) as InstanceGroup;

  it('counts firing alerts no silence suppresses', () => {
    expect(alerts.map((alert) => alert.serviceName || alert.nodeId)).toEqual([
      'database-3-instance-1-reader',
      'mysql-on-premises',
      'database-3-instance-1',
      '',
    ]);
    expect(getFiringAlerts(ALERT_RULES)).toHaveLength(5);
  });

  it('gathers the alerts of a group, with the worst severity', () => {
    const database3 = alertsOf('database-3').alerts ?? [];

    expect(database3).toHaveLength(2);
    expect(getAlertsBand(database3)).toBe('critical');
    expect(alertsOf('database-2').alerts).toEqual([]);
    expect(getAlertsBand([])).toBeUndefined();
  });

  it('gives an alert without service or node name to no instance', () => {
    const stale = withAlerts.find((group) =>
      group.instances.some(({ nodeName }) => !nodeName)
    ) as InstanceGroup;

    expect(stale.alerts).toEqual([]);
  });

  it('matches an alert to an instance by service or node name', () => {
    expect(
      alertsOf('database-3').instances.map(({ serviceName, alerts }) => [
        serviceName,
        alerts?.map((alert) => alert.alertName),
      ])
    ).toEqual([
      ['database-3-instance-1', ['Node CPU']],
      ['database-3-instance-1-reader', ['MySQL down']],
    ]);
  });

  it('links an alert to its rule and a group to its active alerts', () => {
    const queryOf = (url?: string) =>
      Object.fromEntries(new URL(url ?? '', 'http://pmm').searchParams);

    const [firstAlert] = alertsOf('database-3').alerts ?? [];

    expect(getAlertRuleUrl(firstAlert)).toBe(
      '/graph/alerting/grafana/mysql-down/view?tab=instances'
    );
    expect(
      getAlertGroupsUrl(alertsOf('database-3').instances)?.split('?')[0]
    ).toBe('/graph/alerting/groups');
    expect(
      queryOf(getAlertGroupsUrl(alertsOf('database-3').instances))
    ).toEqual({
      queryString:
        'node_name=~"database-3-instance-1|database-3-instance-1-reader"',
      alertState: 'active',
    });
    expect(
      queryOf(getAlertGroupsUrl([instance({ nodeName: 'db.1' })])).queryString
    ).toBe('node_name=~"db\\\\.1"');
    expect(getAlertGroupsUrl([instance({ nodeName: '' })])).toBeUndefined();
  });

  it('leaves the counts unknown without alert data', () => {
    expect(addAlerts(groups)[0].alerts).toBeUndefined();
    expect(addAlerts(groups)[0].instances[0].alerts).toBeUndefined();
  });

  it('sorts by alert count, groups with unknown alerts last', () => {
    const names = (sorted: InstanceGroup[]) => sorted.map(({ name }) => name);

    expect(
      names(sortGroups(withAlerts, [{ id: 'alerts', desc: true }]))[0]
    ).toBe('database-3');
    expect(
      names(
        sortGroups(addAlerts(groups.slice(0, 2)).concat(withAlerts.slice(2)), [
          { id: 'alerts', desc: true },
        ])
      ).slice(-2)
    ).toEqual(names(groups.slice(0, 2)));
  });
});
