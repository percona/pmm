// PMM's labels of an Amazon RDS service, and the AWS labels of its node.
export interface RdsInstanceLabels {
  serviceName: string;
  nodeName: string;
  nodeId: string;
  serviceType: string;
  region: string;
  awsAccountId: string;
  clusterId: string;
  // aws_engine; empty when PMM has no aws_rds_instance_info series for the node.
  awsEngine: string;
  instanceId: string;
  sourceInstanceId: string;
  sourceRegion: string;
}

export type RdsInstanceMetric =
  | 'up'
  | 'innodbReadOnly'
  | 'readOnly'
  | 'inRecovery'
  | 'throughput'
  | 'volumeUsedBytes'
  | 'freeStorageBytes';

// Service metrics are joined by service_name, node metrics (AWS labels, storage) by node_id.
export interface RdsInstanceRow
  extends RdsInstanceLabels, Partial<Record<RdsInstanceMetric, number>> {}
