import { ReactNode } from 'react';
import { CellProps } from '@grafana/ui';

export interface AuroraTreeOptions {
  upField: string;
  readOnlyField: string;
  qpsField: string;
  storageField: string;
}

export type InstanceRole = 'writer' | 'reader' | 'unknown';

export interface InstanceRow {
  region: string;
  awsAccountId: string;
  clusterId: string;
  serviceName: string;
  up?: number;
  readOnly?: number;
  qps?: number;
  storageBytes?: number;
}

export interface AuroraInstance {
  serviceName: string;
  role: InstanceRole;
  up?: number;
  qps?: number;
}

export interface AuroraCluster {
  key: string;
  region: string;
  awsAccountId: string;
  clusterId: string;
  instances: AuroraInstance[];
  monitoredInstances: number;
  exportersConnected: number;
  writers: number;
  qps?: number;
  storageBytes?: number;
}

export interface TableColumn<T extends object> {
  id: keyof T & string;
  header: string;
  cell?: (props: CellProps<T>) => ReactNode;
  sortType?: 'alphanumeric' | 'number';
}
