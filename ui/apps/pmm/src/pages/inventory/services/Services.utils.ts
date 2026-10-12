import type { ManagedService } from 'types/services.types';
import { getAgentsMonitoringStatus } from '../Inventory.utils';
import { Messages } from './Services.messages';
import type {
  ClusterRow,
  ServiceRow,
  ServicesTableRow,
} from './Services.types';

export const isClusterRow = (row: ServicesTableRow): row is ClusterRow =>
  'clusterId' in row;

export const getServicesRowId = (row: ServicesTableRow) =>
  isClusterRow(row) ? row.clusterId : row.serviceId;

export const toServiceRows = (services: ManagedService[]): ServiceRow[] =>
  services
    .map((service) => ({
      ...service,
      // Older clients set the cluster as a custom label
      cluster: service.cluster || service.customLabels.cluster || '',
      agentsStatus: getAgentsMonitoringStatus(service.agents),
    }))
    .sort((a, b) => a.serviceName.localeCompare(b.serviceName));

// Non-clustered services come first, then the clusters by name
export const toClusterRows = (services: ServiceRow[]): ClusterRow[] => {
  const clusters = new Map<string, ServiceRow[]>();

  for (const service of services) {
    clusters.set(service.cluster, [
      ...(clusters.get(service.cluster) ?? []),
      service,
    ]);
  }

  return [...clusters.entries()]
    .sort(([a], [b]) => (a && b ? a.localeCompare(b) : a ? 1 : -1))
    .map(([name, clusterServices]) => ({
      clusterId: `cluster:${name}`,
      name: name || Messages.nonClustered,
      serviceType: clusterServices[0].serviceType,
      services: clusterServices,
    }));
};
