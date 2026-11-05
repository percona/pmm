import {
  dateTime,
  FieldType,
  TimeRange,
  toDataFrame,
} from '@grafana/data';

const GIB = 1024 * 1024 * 1024;

interface MockInfo {
  account: string;
  engine: string;
  instance: string;
  cluster?: string;
  source?: string;
  sourceRegion?: string;
}

interface MockInstance {
  service: string;
  serviceType: 'mysql' | 'postgresql';
  region: string;
  info?: MockInfo;
  // Metric values by query refId; a missing one means no series.
  values: Partial<Record<'A' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H', number>>;
}

const instances: MockInstance[] = [
  {
    service: 'database-2-instance-1',
    serviceType: 'mysql',
    region: 'eu-north-1',
    info: {
      account: '111111111111', engine: 'aurora-mysql', instance: 'database-2-instance-1', cluster: 'database-2',
    },
    values: {
      A: 1, C: 0, D: 0, F: 10, G: 1000,
    },
  },
  {
    service: 'database-3-instance-1',
    serviceType: 'mysql',
    region: 'eu-north-1',
    info: {
      account: '111111111111', engine: 'aurora-mysql', instance: 'database-3-instance-1', cluster: 'database-3',
    },
    values: {
      A: 1, C: 0, D: 0, F: 20.5, G: 5000,
    },
  },
  {
    service: 'database-3-instance-1-reader',
    serviceType: 'mysql',
    region: 'eu-north-1',
    info: {
      account: '111111111111', engine: 'aurora-mysql', instance: 'database-3-instance-1-reader', cluster: 'database-3',
    },
    values: {
      A: 1, C: 1, D: 0, F: 4.25, G: 5000,
    },
  },
  {
    service: 'database-4-instance-1',
    serviceType: 'postgresql',
    region: 'eu-north-1',
    info: {
      account: '111111111111', engine: 'aurora-postgresql', instance: 'database-4-instance-1', cluster: 'database-4',
    },
    values: {
      A: 1, E: 0, F: 15, G: 3000,
    },
  },
  {
    service: 'database-4-instance-1-reader',
    serviceType: 'postgresql',
    region: 'eu-north-1',
    info: {
      account: '111111111111',
      engine: 'aurora-postgresql',
      instance: 'database-4-instance-1-reader',
      cluster: 'database-4',
    },
    values: { A: 1, E: 1, F: 2 },
  },
  {
    service: 'rds-mysql-1',
    serviceType: 'mysql',
    region: 'us-east-2',
    info: { account: '222222222222', engine: 'mysql', instance: 'rds-mysql-1' },
    values: {
      A: 1, C: 0, D: 0, F: 30, H: 20 * GIB,
    },
  },
  {
    service: 'rds-mysql-1-replica',
    serviceType: 'mysql',
    region: 'us-east-2',
    info: {
      account: '222222222222',
      engine: 'mysql',
      instance: 'rds-mysql-1-replica',
      source: 'rds-mysql-1',
      sourceRegion: 'us-east-2',
    },
    values: {
      A: 1, C: 0, D: 1, F: 5, H: 18 * GIB,
    },
  },
  {
    // A cross-region replica whose service name is not its DB instance identifier.
    service: 'rds-mysql-1-west',
    serviceType: 'mysql',
    region: 'us-west-2',
    info: {
      account: '222222222222',
      engine: 'mysql',
      instance: 'rds-mysql-1-replica-west',
      source: 'rds-mysql-1',
      sourceRegion: 'us-east-2',
    },
    values: {
      A: 1, C: 0, D: 1, F: 1, H: 25 * GIB,
    },
  },
  {
    service: 'rds-pg-1',
    serviceType: 'postgresql',
    region: 'us-east-2',
    info: { account: '222222222222', engine: 'postgres', instance: 'rds-pg-1' },
    values: {
      A: 1, E: 0, F: 7.5, H: 50 * GIB,
    },
  },
  {
    service: 'rds-mariadb-1',
    serviceType: 'mysql',
    region: 'us-east-2',
    info: { account: '222222222222', engine: 'mariadb', instance: 'rds-mariadb-1' },
    values: {
      A: 1, C: 0, D: 0, F: 3, H: 10 * GIB,
    },
  },
  {
    service: 'aurora-stale',
    serviceType: 'mysql',
    region: 'eu-north-1',
    values: { A: 0 },
  },
  {
    service: 'legacy-pg',
    serviceType: 'postgresql',
    region: 'eu-north-1',
    values: { A: 1, E: 0, F: 1 },
  },
];

const nodeId = (instance: MockInstance) => `node-${instance.service}`;

const withInfo = instances.filter((instance) => instance.info);

const timeField = (rows: unknown[]) => ({ name: 'Time', type: FieldType.time, values: rows.map(() => 1700000000000) });

const stringField = <T>(name: string, rows: T[], value: (row: T) => string) => ({
  name,
  type: FieldType.string,
  values: rows.map(value),
});

const serviceLabels = (rows: MockInstance[]) => [
  stringField('service_name', rows, (row) => row.service),
  // RDS-added instances have node_name equal to service_name; aurora-stale has none.
  stringField('node_name', rows, (row) => (row.service === 'aurora-stale' ? '' : row.service)),
  stringField('node_id', rows, nodeId),
  stringField('service_type', rows, (row) => row.serviceType),
  stringField('region', rows, (row) => row.region),
];

const infoLabels = (rows: MockInstance[]) => [
  stringField('region', rows, (row) => row.region),
  stringField('aws_account_id', rows, (row) => row.info?.account ?? ''),
  stringField('aws_cluster_id', rows, (row) => row.info?.cluster ?? ''),
  stringField('aws_engine', rows, (row) => row.info?.engine ?? ''),
  stringField('instance', rows, (row) => row.info?.instance ?? ''),
  stringField('aws_source_instance_id', rows, (row) => row.info?.source ?? ''),
  stringField('aws_source_region', rows, (row) => row.info?.sourceRegion ?? ''),
];

const valueField = (refId: keyof MockInstance['values'], rows: MockInstance[]) => ({
  name: `Value #${refId}`,
  type: FieldType.number,
  values: rows.map((row) => row.values[refId] ?? null),
});

const queryFrame = (
  refId: keyof MockInstance['values'] | 'B',
  rows: MockInstance[],
  labels: (present: MockInstance[]) => Array<ReturnType<typeof stringField>>,
) => toDataFrame({
  refId,
  fields: [
    timeField(rows),
    ...labels(rows),
    refId === 'B'
      ? { name: 'Value #B', type: FieldType.number, values: rows.map(() => 1) }
      : valueField(refId, rows),
  ],
});

const has = (refId: keyof MockInstance['values']) => instances.filter((row) => row.values[refId] !== undefined);

const nodeKey = (rows: MockInstance[]) => [stringField('node_id', rows, nodeId)];

// One frame per query, as Prometheus returns them without a transformation.
export const queryFrames = [
  queryFrame('A', has('A'), serviceLabels),
  queryFrame('B', withInfo, (rows) => [...nodeKey(rows), ...infoLabels(rows)]),
  ...(['C', 'D', 'E', 'F'] as const).map((refId) => queryFrame(
    refId,
    has(refId),
    (rows) => [stringField('service_name', rows, (row) => row.service)],
  )),
  ...(['G', 'H'] as const).map((refId) => queryFrame(refId, has(refId), nodeKey)),
];

// Every label and value in one row per service.
export const mergedFrame = toDataFrame({
  fields: [
    timeField(instances),
    ...serviceLabels(instances),
    ...infoLabels(instances).filter((field) => field.name !== 'region'),
    ...(['A', 'C', 'D', 'E', 'F', 'G', 'H'] as const).map((refId) => valueField(refId, instances)),
  ],
});

export const GIB_BYTES = GIB;

export const timeRange: TimeRange = {
  from: dateTime(1700000000000),
  to: dateTime(1700043200000),
  raw: { from: 'now-12h', to: 'now' },
};
