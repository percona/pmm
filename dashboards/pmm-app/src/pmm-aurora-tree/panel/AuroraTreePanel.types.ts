import { ReactNode } from 'react';
import { CellProps } from '@grafana/ui';

export interface AuroraTreeOptions {
  upField: string;
  innodbReadOnlyField: string;
  readOnlyField: string;
  inRecoveryField: string;
  throughputField: string;
  volumeUsedField: string;
  freeStorageField: string;
}

export type EngineId = 'aurora-mysql' | 'aurora-postgresql' | 'mysql' | 'mariadb' | 'postgres';

export type EngineFamily = 'mysql' | 'postgresql';

export interface EngineInfo {
  label: string;
  family: EngineFamily;
  aurora: boolean;
}

export type InstanceRole = 'writer' | 'reader' | 'unknown';

export type GroupKind = 'cluster' | 'replication' | 'instance' | 'unknown';

export type GroupType = 'aurora-cluster' | 'multi-az-cluster' | 'replication' | 'instance' | 'unavailable';

export interface InstanceLabels {
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

export type InstanceMetric = 'up'
| 'innodbReadOnly'
| 'readOnly'
| 'inRecovery'
| 'throughput'
| 'volumeUsedBytes'
| 'freeStorageBytes';

// Service metrics are joined by service_name, node metrics (AWS labels, storage) by node_id.
export interface InstanceRow extends InstanceLabels, Partial<Record<InstanceMetric, number>> {}

export interface Storage {
  kind: 'used' | 'free';
  bytes: number;
  // More than one instance reported free storage, so bytes is the lowest of them.
  lowest?: boolean;
}

export interface RdsInstance {
  serviceName: string;
  nodeName: string;
  instanceId: string;
  region: string;
  engine?: EngineId;
  role: InstanceRole;
  up?: number;
  throughput?: number;
  storage?: Storage;
}

export interface InstanceGroup {
  key: string;
  kind: GroupKind;
  name: string;
  region: string;
  awsAccountId: string;
  engine?: EngineId;
  instances: RdsInstance[];
  monitoredInstances: number;
  exportersConnected: number;
  writers: number;
  unknownRoles: number;
  throughput?: number;
  storage?: Storage;
}

export interface InstanceLink {
  label: string;
  title: string;
  url: string;
}

export interface TableColumn<T extends object> {
  id: keyof T & string;
  header: string;
  cell?: (props: CellProps<T>) => ReactNode;
}

export type SortValue = string | number | undefined;

export interface GroupColumn {
  id: string;
  header: string;
  description?: string;
  render: (group: InstanceGroup) => ReactNode;
  sortValue: (group: InstanceGroup) => SortValue;
}

export interface GroupSort {
  id: string;
  desc: boolean;
}

export interface GroupSelection {
  regions: string[];
  accounts: string[];
  engines: string[];
  types: string[];
  groups: string[];
}

export interface GroupFilters extends GroupSelection {
  query: string;
}

export interface FilteredGroups {
  groups: InstanceGroup[];
  autoExpanded: Set<string>;
}
