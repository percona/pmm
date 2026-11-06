import type { AlertSeverityBand } from 'types/alerting.types';

export type EngineId =
  | 'aurora-mysql'
  | 'aurora-postgresql'
  | 'mysql'
  | 'mariadb'
  | 'postgres';

export type EngineFamily = 'mysql' | 'postgresql';

export interface EngineInfo {
  label: string;
  family: EngineFamily;
  aurora: boolean;
}

export type InstanceRole = 'writer' | 'reader' | 'unknown';

export type GroupKind = 'cluster' | 'replication' | 'instance' | 'unknown';

export type GroupType =
  | 'aurora-cluster'
  | 'multi-az-cluster'
  | 'replication'
  | 'instance'
  | 'unavailable';

export interface Storage {
  kind: 'used' | 'free';
  bytes: number;
  // More than one instance reported free storage, so bytes is the lowest of them.
  lowest?: boolean;
}

export interface AlertSummary {
  count: number;
  // Severity of the most severe alert; undefined when none has a known severity.
  band?: AlertSeverityBand;
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
  // Undefined while the alerts are unknown: not loaded or not readable.
  alerts?: AlertSummary;
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
  // Undefined while the alerts are unknown: not loaded or not readable.
  alerts?: AlertSummary;
}

export interface InstanceLink {
  label: string;
  title: string;
  url: string;
}

export interface InstanceLinks {
  service: InstanceLink;
  dashboards: InstanceLink[];
}

export type WritersState = 'good' | 'warning' | 'unknown';

export type GroupColumnId =
  | 'region'
  | 'awsAccountId'
  | 'name'
  | 'type'
  | 'engine'
  | 'alerts'
  | 'instances'
  | 'writers'
  | 'throughput'
  | 'storage';

export type SortValue = string | number | undefined;

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

export interface FilterOption {
  label: string;
  value: string;
  // The regions the value occurs in.
  description?: string;
}
