import type { ManagedService, ManagedServiceType } from 'types/services.types';
import type { MonitoringStatus } from '../Inventory.types';

export interface ServiceRow extends ManagedService {
  agentsStatus: MonitoringStatus;
}

// Groups a cluster's services as sub rows in the cluster view
export interface ClusterRow {
  clusterId: string;
  name: string;
  serviceType?: ManagedServiceType;
  services: ServiceRow[];
}

export type ServicesTableRow = ServiceRow | ClusterRow;
