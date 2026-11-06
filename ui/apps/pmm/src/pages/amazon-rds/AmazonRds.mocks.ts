import type { RdsInstanceRow } from 'types/rds.types';

export const GIB = 1024 * 1024 * 1024;

const AURORA_ACCOUNT = '111111111111';
const RDS_ACCOUNT = '222222222222';

export const rdsRow = (
  serviceName: string,
  values: Partial<RdsInstanceRow> = {}
): RdsInstanceRow => ({
  serviceName,
  // RDS-added instances have node_name equal to service_name.
  nodeName: serviceName,
  nodeId: `node-${serviceName}`,
  serviceType: 'mysql',
  region: 'eu-north-1',
  awsAccountId: '',
  clusterId: '',
  awsEngine: '',
  instanceId: '',
  sourceInstanceId: '',
  sourceRegion: '',
  ...values,
});

const aurora = (
  serviceName: string,
  clusterId: string,
  values: Partial<RdsInstanceRow>
) =>
  rdsRow(serviceName, {
    awsAccountId: AURORA_ACCOUNT,
    awsEngine: 'aurora-mysql',
    instanceId: serviceName,
    clusterId,
    ...values,
  });

const rds = (serviceName: string, values: Partial<RdsInstanceRow>) =>
  rdsRow(serviceName, {
    region: 'us-east-2',
    awsAccountId: RDS_ACCOUNT,
    awsEngine: 'mysql',
    instanceId: serviceName,
    ...values,
  });

// Three Aurora clusters, a replication group with a cross-region replica, two standalone
// instances, and two services PMM has no AWS details for.
export const RDS_ROWS: RdsInstanceRow[] = [
  aurora('database-2-instance-1', 'database-2', {
    up: 1,
    innodbReadOnly: 0,
    readOnly: 0,
    throughput: 10,
    volumeUsedBytes: 1000,
  }),
  aurora('database-3-instance-1', 'database-3', {
    up: 1,
    innodbReadOnly: 0,
    readOnly: 0,
    throughput: 20.5,
    volumeUsedBytes: 5000,
  }),
  aurora('database-3-instance-1-reader', 'database-3', {
    up: 1,
    innodbReadOnly: 1,
    readOnly: 0,
    throughput: 4.25,
    volumeUsedBytes: 5000,
  }),
  aurora('database-4-instance-1', 'database-4', {
    serviceType: 'postgresql',
    awsEngine: 'aurora-postgresql',
    up: 1,
    inRecovery: 0,
    throughput: 15,
    volumeUsedBytes: 3000,
  }),
  aurora('database-4-instance-1-reader', 'database-4', {
    serviceType: 'postgresql',
    awsEngine: 'aurora-postgresql',
    up: 1,
    inRecovery: 1,
    throughput: 2,
  }),
  rds('rds-mysql-1', {
    up: 1,
    innodbReadOnly: 0,
    readOnly: 0,
    throughput: 30,
    freeStorageBytes: 20 * GIB,
  }),
  rds('rds-mysql-1-replica', {
    sourceInstanceId: 'rds-mysql-1',
    sourceRegion: 'us-east-2',
    up: 1,
    innodbReadOnly: 0,
    readOnly: 1,
    throughput: 5,
    freeStorageBytes: 18 * GIB,
  }),
  // A cross-region replica whose service name is not its DB instance identifier.
  rds('rds-mysql-1-west', {
    region: 'us-west-2',
    instanceId: 'rds-mysql-1-replica-west',
    sourceInstanceId: 'rds-mysql-1',
    sourceRegion: 'us-east-2',
    up: 1,
    innodbReadOnly: 0,
    readOnly: 1,
    throughput: 1,
    freeStorageBytes: 25 * GIB,
  }),
  rds('rds-pg-1', {
    serviceType: 'postgresql',
    awsEngine: 'postgres',
    up: 1,
    inRecovery: 0,
    throughput: 7.5,
    freeStorageBytes: 50 * GIB,
  }),
  rds('rds-mariadb-1', {
    awsEngine: 'mariadb',
    up: 1,
    innodbReadOnly: 0,
    readOnly: 0,
    throughput: 3,
    freeStorageBytes: 10 * GIB,
  }),
  rdsRow('aurora-stale', { nodeName: '', up: 0 }),
  rdsRow('legacy-pg', {
    serviceType: 'postgresql',
    up: 1,
    inRecovery: 0,
    throughput: 1,
  }),
];
