import type { InstantSample } from 'types/metrics.types';
import type {
  RdsInstanceLabels,
  RdsInstanceMetric,
  RdsInstanceRow,
} from 'types/rds.types';
import { queryInstant } from './metrics';

const LABELS: Record<keyof RdsInstanceLabels, string> = {
  serviceName: 'service_name',
  nodeName: 'node_name',
  nodeId: 'node_id',
  serviceType: 'service_type',
  region: 'region',
  awsAccountId: 'aws_account_id',
  clusterId: 'aws_cluster_id',
  awsEngine: 'aws_engine',
  // aws_rds_instance_info carries the DB instance identifier as instance.
  instanceId: 'instance',
  sourceInstanceId: 'aws_source_instance_id',
  sourceRegion: 'aws_source_region',
};

const LABEL_KEYS = Object.keys(LABELS) as Array<keyof RdsInstanceLabels>;

// Each refId names the metric its query reads; info carries the AWS labels of the node. A
// service whose exporter stopped reporting stays for an hour as down, and so do its labels.
export const RDS_QUERIES: Record<RdsInstanceMetric | 'info', string> = {
  up: `max by (service_name, node_name, node_id, service_type, region) ({__name__=~"mysql_up|pg_up", node_type="remote_rds"})
or
(
  max by (service_name, node_name, node_id, service_type, region) (last_over_time({__name__=~"mysql_up|pg_up", node_type="remote_rds"}[1h]))
  unless on (service_name) {__name__=~"mysql_up|pg_up", node_type="remote_rds"}
) * 0`,
  info: `max by (node_id, region, aws_account_id, aws_cluster_id, aws_engine, instance, aws_source_instance_id, aws_source_region) (
  aws_rds_instance_info
  or
  (last_over_time(aws_rds_instance_info[1h]) unless on (node_id) aws_rds_instance_info)
)`,
  innodbReadOnly: `max by (service_name) (mysql_global_variables_innodb_read_only{node_type="remote_rds"})`,
  readOnly: `max by (service_name) (mysql_global_variables_read_only{node_type="remote_rds"})`,
  inRecovery: `max by (service_name) (pg_replication_is_replica{node_type="remote_rds"})
or
max by (service_name) (pg_settings_in_hot_standby{node_type="remote_rds"})`,
  throughput: `sum by (service_name) (rate(mysql_global_status_queries{node_type="remote_rds"}[$__rate_interval]))
or
(
  sum by (service_name) (rate(pg_stat_database_xact_commit{node_type="remote_rds"}[$__rate_interval]))
  + sum by (service_name) (rate(pg_stat_database_xact_rollback{node_type="remote_rds"}[$__rate_interval]))
)`,
  volumeUsedBytes: `max by (node_id) (aws_rds_volume_bytes_used_average)`,
  freeStorageBytes: `max by (node_id) (aws_rds_free_storage_space_average)
or
max by (node_id) (node_filesystem_free_bytes{node_type="remote_rds", mountpoint=""})`,
};

const isMetric = (refId: string): refId is RdsInstanceMetric =>
  refId !== 'info' && Object.keys(RDS_QUERIES).includes(refId);

const METRIC_KEYS = Object.keys(RDS_QUERIES).filter(isMetric);

const emptyRow = (): RdsInstanceRow => ({
  serviceName: '',
  nodeName: '',
  nodeId: '',
  serviceType: '',
  region: '',
  awsAccountId: '',
  clusterId: '',
  awsEngine: '',
  instanceId: '',
  sourceInstanceId: '',
  sourceRegion: '',
});

// Labels keep their first non-empty value, metrics their first defined one.
const fillRow = (row: RdsInstanceRow, from: RdsInstanceRow) => {
  LABEL_KEYS.forEach((key) => {
    row[key] = row[key] || from[key];
  });
  METRIC_KEYS.forEach((key) => {
    row[key] = row[key] ?? from[key];
  });

  return row;
};

// A series with service_name belongs to a service, one with only node_id to its node.
export const readRdsInstances = (
  results: Record<string, InstantSample[]>
): RdsInstanceRow[] => {
  const services = new Map<string, RdsInstanceRow>();
  const nodes = new Map<string, RdsInstanceRow>();

  Object.entries(results).forEach(([refId, samples]) => {
    samples.forEach(({ labels, value }) => {
      const serviceName = labels[LABELS.serviceName];
      const key = serviceName || labels[LABELS.nodeId];
      const rows = serviceName ? services : nodes;

      if (!key) {
        return;
      }

      const row = rows.get(key) ?? emptyRow();

      LABEL_KEYS.forEach((label) => {
        row[label] = row[label] || (labels[LABELS[label]] ?? '');
      });

      if (isMetric(refId) && value !== undefined) {
        row[refId] = value;
      }

      rows.set(key, row);
    });
  });

  return [...services.values()].map((service) => {
    const node = nodes.get(service.nodeId);

    return node ? fillRow(service, node) : service;
  });
};

export const getRdsInstances = async (datasourceUid: string) =>
  readRdsInstances(
    await queryInstant(
      datasourceUid,
      Object.entries(RDS_QUERIES).map(([refId, expr]) => ({ refId, expr }))
    )
  );
