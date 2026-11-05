import { DEFAULT_OPTIONS } from './AuroraTreePanel.constants';
import { GIB_BYTES, mergedFrame, queryFrames } from './AuroraTreePanel.mocks';
import {
  compareSortValues,
  filterGroups,
  formatStorage,
  formatThroughput,
  formatValue,
  getAccountOptions,
  getEngine,
  getEngineOptions,
  getGroupOptions,
  getGroupType,
  getTypeOptions,
  getInstanceDetails,
  getInstanceLinks,
  getRegionOptions,
  getRole,
  getWriters,
  groupInstances,
  narrowSelection,
  readFilters,
  readInstances,
  toUrlQuery,
} from './AuroraTreePanel.tools';
import {
  GroupFilters,
  InstanceGroup,
  InstanceRow,
  RdsInstance,
} from './AuroraTreePanel.types';

jest.mock('@grafana/runtime', () => ({
  config: { appSubUrl: '/graph' },
}));

const groups = groupInstances(readInstances(queryFrames, DEFAULT_OPTIONS));

const findGroup = (name: string, engine?: string) => groups
  .find((group) => group.name === name && (!engine || group.engine === engine)) as InstanceGroup;

const row = (values: Partial<InstanceRow>): InstanceRow => ({
  serviceName: 'svc',
  nodeName: '',
  nodeId: '',
  serviceType: 'mysql',
  region: 'eu-north-1',
  awsAccountId: '111111111111',
  clusterId: '',
  awsEngine: 'mysql',
  instanceId: values.serviceName ?? 'svc',
  sourceInstanceId: '',
  sourceRegion: '',
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

describe('AuroraTreePanel.tools:: grouping', () => {
  it('groups clusters, replication groups and standalone instances, without AWS details last in a region', () => {
    expect(groups.map(({ name, kind, engine }) => [name, kind, engine])).toEqual([
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
    expect(findGroup('database-3').instances.map(({ serviceName, role }) => [serviceName, role])).toEqual([
      ['database-3-instance-1', 'writer'],
      ['database-3-instance-1-reader', 'reader'],
    ]);
  });

  it('takes the volume of an Aurora PostgreSQL cluster from any instance that reports it', () => {
    const database4 = findGroup('database-4');

    expect(database4).toMatchObject({ writers: 1, throughput: 17, storage: { kind: 'used', bytes: 3000 } });
    expect(database4.instances.map(({ role, storage }) => [role, storage])).toEqual([
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
      storage: { kind: 'free', bytes: 18 * GIB_BYTES, lowest: true },
    });
    expect(replication.instances.map((item) => [item.serviceName, item.role, item.region])).toEqual([
      ['rds-mysql-1', 'writer', 'us-east-2'],
      ['rds-mysql-1-replica', 'reader', 'us-east-2'],
      ['rds-mysql-1-west', 'reader', 'us-west-2'],
    ]);
  });

  it('keeps a standalone instance as its own group with its free storage', () => {
    expect(findGroup('rds-pg-1')).toMatchObject({
      monitoredInstances: 1,
      writers: 1,
      storage: { kind: 'free', bytes: 50 * GIB_BYTES },
    });
    expect(findGroup('rds-pg-1').instances[0].storage).toEqual({ kind: 'free', bytes: 50 * GIB_BYTES });
  });

  it('puts instances without AWS details in one group per region and engine', () => {
    const mysql = findGroup('AWS info unavailable', 'mysql');

    expect(mysql).toMatchObject({
      awsAccountId: '', exportersConnected: 0, writers: 0, unknownRoles: 1,
    });
    expect(mysql.instances).toEqual([{
      serviceName: 'aurora-stale',
      nodeName: '',
      instanceId: '',
      region: 'eu-north-1',
      engine: 'mysql',
      role: 'unknown',
      up: 0,
      throughput: undefined,
      storage: undefined,
    }]);
    expect(findGroup('AWS info unavailable', 'postgres').instances.map(({ serviceName }) => serviceName))
      .toEqual(['legacy-pg']);
  });

  it('joins one frame per query the same way as one merged frame', () => {
    expect(groupInstances(readInstances([mergedFrame], DEFAULT_OPTIONS))).toEqual(groups);
  });

  it('reads the value fields named in the options', () => {
    const options = { ...DEFAULT_OPTIONS, innodbReadOnlyField: 'Value #A', upField: 'Value #C' };
    const database2 = groupInstances(readInstances(queryFrames, options))
      .find((group) => group.name === 'database-2') as InstanceGroup;

    expect(database2.instances[0]).toMatchObject({ role: 'reader', up: 0 });
  });

  it('joins cascaded replicas to the root primary', () => {
    const [group] = groupInstances([
      row({ serviceName: 'primary' }),
      row({ serviceName: 'replica-1', sourceInstanceId: 'primary', sourceRegion: 'eu-north-1' }),
      row({ serviceName: 'replica-2', sourceInstanceId: 'replica-1', sourceRegion: 'eu-north-1' }),
    ]);

    expect(group).toMatchObject({ kind: 'replication', name: 'primary', monitoredInstances: 3 });
  });

  it('names a replication group after a primary PMM does not monitor', () => {
    const replicaGroups = groupInstances([
      row({ serviceName: 'replica-1', sourceInstanceId: 'primary', sourceRegion: 'eu-north-1' }),
      row({ serviceName: 'replica-2', sourceInstanceId: 'primary' }),
    ]);

    expect(replicaGroups).toHaveLength(1);
    expect(replicaGroups[0]).toMatchObject({ kind: 'replication', name: 'primary', monitoredInstances: 2 });
  });

  it('groups a Multi-AZ DB cluster by cluster ID with its free storage', () => {
    const [group] = groupInstances([
      row({ serviceName: 'multi-az-1', clusterId: 'multi-az', freeStorageBytes: 30 }),
      row({ serviceName: 'multi-az-2', clusterId: 'multi-az', freeStorageBytes: 20 }),
    ]);

    expect(group).toMatchObject({
      kind: 'cluster', name: 'multi-az', storage: { kind: 'free', bytes: 20, lowest: true },
    });
  });
});

describe('AuroraTreePanel.tools:: roles and engines', () => {
  it('reads the engine from aws_engine, falling back to the service type', () => {
    expect(getEngine(row({ awsEngine: 'aurora-postgresql' }))).toBe('aurora-postgresql');
    expect(getEngine(row({ awsEngine: 'aurora' }))).toBe('aurora-mysql');
    expect(getEngine(row({ awsEngine: '', serviceType: 'postgresql' }))).toBe('postgres');
    expect(getEngine(row({ awsEngine: '', serviceType: 'mysql' }))).toBe('mysql');
    expect(getEngine(row({ awsEngine: '', serviceType: '' }))).toBeUndefined();
  });

  it('takes the Aurora MySQL role from innodb_read_only', () => {
    const aurora = { awsEngine: 'aurora-mysql', readOnly: 0 };

    expect(getRole(row({ ...aurora, innodbReadOnly: 0 }), 'aurora-mysql')).toBe('writer');
    expect(getRole(row({ ...aurora, innodbReadOnly: 1 }), 'aurora-mysql')).toBe('reader');
    expect(getRole(row(aurora), 'aurora-mysql')).toBe('unknown');
  });

  it('takes the RDS MySQL and MariaDB role from read_only', () => {
    expect(getRole(row({ innodbReadOnly: 0, readOnly: 1 }), 'mysql')).toBe('reader');
    expect(getRole(row({ innodbReadOnly: 0, readOnly: 0 }), 'mysql')).toBe('writer');
    expect(getRole(row({ awsEngine: 'mariadb', readOnly: 1 }), 'mariadb')).toBe('reader');
    expect(getRole(row({ innodbReadOnly: 0 }), 'mysql')).toBe('unknown');
  });

  it('takes the PostgreSQL role from recovery, for Aurora and RDS', () => {
    expect(getRole(row({ awsEngine: 'postgres', inRecovery: 0 }), 'postgres')).toBe('writer');
    expect(getRole(row({ awsEngine: 'aurora-postgresql', inRecovery: 1 }), 'aurora-postgresql')).toBe('reader');
    expect(getRole(row({ awsEngine: 'aurora-postgresql', readOnly: 0 }), 'aurora-postgresql')).toBe('unknown');
  });

  it('marks a MySQL reader by either variable without AWS details', () => {
    const noInfo = { awsEngine: '' };

    expect(getRole(row({ ...noInfo, innodbReadOnly: 1, readOnly: 0 }), 'mysql')).toBe('reader');
    expect(getRole(row({ ...noInfo, innodbReadOnly: 0, readOnly: 1 }), 'mysql')).toBe('reader');
    expect(getRole(row({ ...noInfo, innodbReadOnly: 0, readOnly: 0 }), 'mysql')).toBe('writer');
    expect(getRole(row(noInfo), 'mysql')).toBe('unknown');
  });

  it('describes the writers of a group', () => {
    const group = (writers: number, unknownRoles = 0) => ({ writers, unknownRoles } as InstanceGroup);

    expect(getWriters(group(1, 1))).toEqual({ text: '1 writer', state: 'good' });
    expect(getWriters(group(2))).toEqual({ text: '2 writers', state: 'warning' });
    expect(getWriters(group(0))).toEqual({ text: 'no writer', state: 'warning' });
    expect(getWriters(group(0, 2))).toEqual({ text: 'unknown', state: 'unknown' });
  });
});

describe('AuroraTreePanel.tools:: formatting and links', () => {
  it('formats values with Grafana units', () => {
    expect(formatValue(undefined, 'qps', 2)).toBe('-');
    expect(formatValue(5 * GIB_BYTES, 'bytes')).toBe('5 GiB');
  });

  it('formats throughput as queries/s for MySQL and transactions/s for PostgreSQL', () => {
    expect(formatThroughput(24.75, 'aurora-mysql')).toBe('24.75 qps');
    expect(formatThroughput(3, 'mariadb')).toBe('3.00 qps');
    expect(formatThroughput(17, 'aurora-postgresql')).toBe('17.00 tps');
    expect(formatThroughput(undefined, 'postgres')).toBe('-');
  });

  it('labels Aurora volume as used and other storage as free', () => {
    expect(formatStorage({ kind: 'used', bytes: 765 * 1024 * 1024 })).toBe('765 MiB used');
    expect(formatStorage({ kind: 'free', bytes: 18 * GIB_BYTES })).toBe('18 GiB free');
    expect(formatStorage({ kind: 'free', bytes: 18 * GIB_BYTES, lowest: true })).toBe('18 GiB free (min)');
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
      const { service, dashboards } = getInstanceLinks(instance({ engine, serviceName: 'db 1' }));

      expect(service).toEqual({
        label: 'db 1',
        title: 'MySQL Instance Summary for db 1',
        url: '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=db%201',
      });
      expect(dashboards.map(({ label }) => label)).toEqual(['OS']);
    },
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
    },
  );

  it('links a node to its Node Summary, falling back to the service name', () => {
    const [os] = getInstanceLinks(instance({ engine: 'postgres', nodeName: '' })).dashboards;

    expect(os.url).toBe('/graph/d/node-instance-summary/node-summary?var-node_name=db-1');
  });

  it('shows the instance ID and region only where they differ from the service and group', () => {
    const replication = findGroup('rds-mysql-1');
    const [primary, , west] = replication.instances;

    expect(getInstanceDetails(primary, replication)).toBe('');
    expect(getInstanceDetails(west, replication)).toBe('rds-mysql-1-replica-west, us-west-2');
    expect(getInstanceDetails(findGroup('AWS info unavailable').instances[0], replication)).toBe('eu-north-1');
  });
});

describe('AuroraTreePanel.tools:: filters', () => {
  const noFilters: GroupFilters = {
    query: '', regions: [], accounts: [], engines: [], types: [], groups: [],
  };
  const values = (options: Array<{ value: string }>) => options.map(({ value }) => value);
  const filter = (filters: Partial<GroupFilters>) => filterGroups(groups, { ...noFilters, ...filters });
  const names = (filters: Partial<GroupFilters>) => filter(filters).groups.map(({ name }) => name);

  it('keeps every group without filters', () => {
    expect(filter({})).toEqual({ groups, autoExpanded: new Set() });
  });

  it('searches group name, account, region and engine case-insensitively, keeping all instances', () => {
    expect(names({ query: 'DATABASE-3' })).toEqual(['database-3']);
    expect(names({ query: '2222' })).toEqual(['rds-mariadb-1', 'rds-mysql-1', 'rds-pg-1']);
    expect(names({ query: 'mariadb' })).toEqual(['rds-mariadb-1']);
    expect(names({ query: 'aurora postgresql' })).toEqual(['database-4']);
    expect(filter({ query: 'rds-mysql-1' }).groups[0].instances).toHaveLength(3);
    expect(filter({ query: 'rds-mysql-1' }).autoExpanded.size).toBe(0);
  });

  it('narrows a group matched only by an instance to those instances and expands it', () => {
    const { groups: matching, autoExpanded } = filter({ query: ' Reader ' });

    expect(matching.map(({ name }) => name)).toEqual(['database-3', 'database-4']);
    expect(matching[0]).toMatchObject({ monitoredInstances: 2 });
    expect(matching[0].instances.map(({ serviceName }) => serviceName)).toEqual(['database-3-instance-1-reader']);
    expect(autoExpanded).toEqual(new Set(matching.map(({ key }) => key)));
  });

  it('finds an instance by its AWS instance ID', () => {
    expect(filter({ query: 'replica-west' }).groups[0].instances.map(({ serviceName }) => serviceName))
      .toEqual(['rds-mysql-1-west']);
  });

  it('filters by engine, alone and combined with the other filters', () => {
    expect(names({ engines: ['aurora-postgresql'] })).toEqual(['database-4']);
    expect(names({ engines: ['postgres'] })).toEqual(['AWS info unavailable', 'rds-pg-1']);
    expect(names({ engines: ['mysql', 'mariadb'], regions: ['us-east-2'] })).toEqual(['rds-mariadb-1', 'rds-mysql-1']);
    expect(names({ engines: ['aurora-mysql'], accounts: ['222222222222'] })).toEqual([]);
  });

  it('combines account and group filters with the search', () => {
    expect(names({ accounts: ['111111111111'] })).toEqual(['database-2', 'database-3', 'database-4']);
    expect(names({ accounts: ['111111111111'], groups: ['database-3'] })).toEqual(['database-3']);
    expect(names({ accounts: ['222222222222'], groups: ['database-3'] })).toEqual([]);
    expect(names({ accounts: ['111111111111', '222222222222'], query: 'instance-1' })).toEqual([
      'database-2',
      'database-3',
      'database-4',
    ]);
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
    expect(getGroupType({ kind: 'cluster', engine: 'postgres' } as InstanceGroup)).toBe('multi-az-cluster');
    expect(getGroupType({ kind: 'cluster', engine: 'mysql' } as InstanceGroup)).toBe('multi-az-cluster');
    expect(getGroupType({ kind: 'cluster', engine: 'aurora-postgresql' } as InstanceGroup)).toBe('aurora-cluster');
  });

  it('filters by type and searches the type text', () => {
    expect(names({ types: ['aurora-cluster'] })).toEqual(['database-2', 'database-3', 'database-4']);
    expect(names({ types: ['instance', 'replication'] })).toEqual(['rds-mariadb-1', 'rds-mysql-1', 'rds-pg-1']);
    expect(names({ types: ['unavailable'] })).toEqual(['AWS info unavailable', 'AWS info unavailable']);
    expect(names({ types: ['aurora-cluster'], engines: ['aurora-postgresql'] })).toEqual(['database-4']);
    expect(names({ query: 'aurora cluster' })).toEqual(['database-2', 'database-3', 'database-4']);
    expect(names({ query: 'replication group' })).toEqual(['rds-mysql-1']);
  });

  it('builds type options from the data and narrows the DB identifier options by type', () => {
    expect(getTypeOptions(groups)).toEqual([
      { label: 'Aurora cluster', value: 'aurora-cluster' },
      { label: 'Instance', value: 'instance' },
      { label: 'Replication group', value: 'replication' },
      { label: 'Unavailable', value: 'unavailable' },
    ]);
    expect(values(getGroupOptions(groups, {
      regions: [], accounts: [], engines: [], types: ['instance', 'replication'],
    }))).toEqual(['rds-mariadb-1', 'rds-mysql-1', 'rds-pg-1']);
    expect(narrowSelection(groups, {
      regions: [], accounts: [], engines: [], types: ['instance'], groups: ['database-3', 'rds-pg-1'],
    }).groups).toEqual(['rds-pg-1']);
  });

  it('finds the groups without AWS details by search and by group filter', () => {
    expect(names({ query: 'unavailable' })).toEqual(['AWS info unavailable', 'AWS info unavailable']);
    expect(names({ groups: ['AWS info unavailable'] })).toEqual(['AWS info unavailable', 'AWS info unavailable']);
    expect(names({ query: 'stale' })).toEqual(['AWS info unavailable']);
  });

  it('returns nothing when no group matches', () => {
    expect(filter({ query: 'no-such-group' })).toEqual({ groups: [], autoExpanded: new Set() });
  });

  it('filters by region, alone and combined with the other filters', () => {
    expect(names({ regions: ['us-east-2'] })).toEqual(['rds-mariadb-1', 'rds-mysql-1', 'rds-pg-1']);
    expect(names({ regions: ['us-east-2'], groups: ['database-3'] })).toEqual([]);
    expect(names({ regions: ['eu-north-1'], query: 'reader' })).toEqual(['database-3', 'database-4']);
  });

  it('builds region, account and engine options from the data', () => {
    expect(getRegionOptions(groups)).toEqual([
      { label: 'eu-north-1', value: 'eu-north-1' },
      { label: 'us-east-2', value: 'us-east-2' },
    ]);
    expect(getAccountOptions(groups, [])).toEqual([
      { label: '111111111111', value: '111111111111', description: 'eu-north-1' },
      { label: '222222222222', value: '222222222222', description: 'us-east-2' },
    ]);
    expect(values(getAccountOptions(groups, ['us-east-2']))).toEqual(['222222222222']);
    expect(getEngineOptions(groups)).toEqual([
      { label: 'Aurora MySQL', value: 'aurora-mysql' },
      { label: 'Aurora PostgreSQL', value: 'aurora-postgresql' },
      { label: 'MariaDB', value: 'mariadb' },
      { label: 'MySQL', value: 'mysql' },
      { label: 'PostgreSQL', value: 'postgres' },
    ]);
  });

  it('builds group options narrowed to the selected regions, accounts and engines, unknown group last', () => {
    const selection = {
      regions: [], accounts: [], engines: [], types: [],
    };

    expect(values(getGroupOptions(groups, selection))).toEqual([
      'database-2',
      'database-3',
      'database-4',
      'rds-mariadb-1',
      'rds-mysql-1',
      'rds-pg-1',
      'AWS info unavailable',
    ]);
    expect(values(getGroupOptions(groups, { ...selection, accounts: ['222222222222'] }))).toEqual([
      'rds-mariadb-1',
      'rds-mysql-1',
      'rds-pg-1',
    ]);
    expect(values(getGroupOptions(groups, { ...selection, engines: ['postgres', 'aurora-postgresql'] }))).toEqual([
      'database-4',
      'rds-pg-1',
      'AWS info unavailable',
    ]);
  });

  it('drops selected accounts and groups the selected regions, accounts and engines no longer offer', () => {
    const selection = {
      regions: [],
      accounts: ['111111111111', '222222222222'],
      engines: [],
      types: [],
      groups: ['database-3', 'rds-mysql-1'],
    };

    expect(narrowSelection(groups, { ...selection, regions: ['us-east-2'] })).toEqual({
      ...selection,
      regions: ['us-east-2'],
      accounts: ['222222222222'],
      groups: ['rds-mysql-1'],
    });
    expect(narrowSelection(groups, { ...selection, engines: ['aurora-mysql'] })).toEqual({
      ...selection,
      engines: ['aurora-mysql'],
      groups: ['database-3'],
    });
    expect(narrowSelection(groups, selection)).toEqual(selection);
  });

  it('reads filters from URL parameters', () => {
    expect(readFilters({})).toEqual(noFilters);
    expect(readFilters({
      'rds-q': 'reader',
      'rds-region': ['eu-north-1', 'us-east-2'],
      'rds-account': '111111111111',
      'rds-engine': 'postgres',
      'rds-type': 'aurora-cluster',
      'rds-id': ['database-2', 'database-3'],
    })).toEqual({
      query: 'reader',
      regions: ['eu-north-1', 'us-east-2'],
      accounts: ['111111111111'],
      engines: ['postgres'],
      types: ['aurora-cluster'],
      groups: ['database-2', 'database-3'],
    });
    expect(readFilters({ 'rds-group': 'database-2' }).groups).toEqual(['database-2']);
    expect(readFilters({ 'rds-group': 'old', 'rds-id': 'new' }).groups).toEqual(['new']);
  });

  it('writes filters to URL parameters, removing empty ones', () => {
    expect(toUrlQuery(noFilters)).toEqual({
      'rds-q': null,
      'rds-region': null,
      'rds-account': null,
      'rds-engine': null,
      'rds-type': null,
      'rds-id': null,
      'rds-group': null,
    });
    expect(toUrlQuery({
      query: 'db', regions: ['r'], accounts: ['1'], engines: ['mysql'], types: ['instance'], groups: ['g'],
    })).toEqual({
      'rds-q': 'db',
      'rds-region': ['r'],
      'rds-account': ['1'],
      'rds-engine': ['mysql'],
      'rds-type': ['instance'],
      'rds-id': ['g'],
      'rds-group': null,
    });
  });

  it('sorts missing values last in both directions', () => {
    expect(compareSortValues(undefined, 1, false)).toBe(1);
    expect(compareSortValues(undefined, 1, true)).toBe(1);
    expect(compareSortValues(1, undefined, true)).toBe(-1);
    expect(compareSortValues(1, 3, false)).toBeLessThan(0);
    expect(compareSortValues(1, 3, true)).toBeGreaterThan(0);
    expect(compareSortValues('b', 'a', false)).toBeGreaterThan(0);
  });
});
