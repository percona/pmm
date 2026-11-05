import {
  dateTime,
  FieldType,
  TimeRange,
  toDataFrame,
} from '@grafana/data';

const instances = [
  ['eu-north-1', '111111111111', 'database-2', 'database-2-instance-1'],
  ['eu-north-1', '111111111111', 'database-3', 'database-3-instance-1'],
  ['eu-north-1', '111111111111', 'database-3', 'database-3-instance-1-reader'],
  ['eu-north-1', '', 'AWS cluster unknown', 'aurora-stale'],
  ['us-east-2', '222222222222', 'pmm-qa-aurora3-mysql', 'pmm-qa-aurora3-mysql-instance-1'],
];

const labelFields = (rows: string[][]) => [
  { name: 'Time', type: FieldType.time, values: rows.map(() => 1700000000000) },
  { name: 'region', type: FieldType.string, values: rows.map((row) => row[0]) },
  { name: 'aws_account_id', type: FieldType.string, values: rows.map((row) => row[1]) },
  { name: 'aws_cluster_id', type: FieldType.string, values: rows.map((row) => row[2]) },
  { name: 'service_name', type: FieldType.string, values: rows.map((row) => row[3]) },
];

// Values per instance, in the order of `instances`; null marks a missing series.
const values: Record<string, Array<number | null>> = {
  'Value #A': [1, 1, 1, 0, 1],
  'Value #B': [0, 0, 1, null, 0],
  'Value #C': [10, 20.5, 4.25, null, 7],
  'Value #D': [1000, 5000, 5000, null, 2000],
};

export const mergedFrame = toDataFrame({
  fields: [
    ...labelFields(instances),
    ...Object.entries(values).map(([name, column]) => ({ name, type: FieldType.number, values: column })),
  ],
});

// One frame per query, as Prometheus returns them before a merge transformation.
export const unmergedFrames = Object.entries(values).map(([name, column]) => {
  const present = instances.filter((_, index) => column[index] !== null);

  return toDataFrame({
    refId: name.replace('Value #', ''),
    fields: [
      ...labelFields(present),
      { name, type: FieldType.number, values: column.filter((value) => value !== null) },
    ],
  });
});

export const timeRange: TimeRange = {
  from: dateTime(1700000000000),
  to: dateTime(1700043200000),
  raw: { from: 'now-12h', to: 'now' },
};
