import { ManagedService } from 'types/services.types';
import { ServiceOption, ClusterSelectionState } from './RealTimeSelectionForm.types';

export type { ServiceOption, ClusterSelectionState };

/**
 * Get the selection state of a cluster
 */
export const getClusterSelectionState = (
  clusterName: string,
  serviceOptions: ServiceOption[],
  selectedServices: ServiceOption[]
): ClusterSelectionState => {
  const servicesInCluster = serviceOptions.filter(
    (option) => option.type === 'service' && option.cluster === clusterName
  );

  if (servicesInCluster.length === 0) {
    return 'none';
  }

  const selectedCount = servicesInCluster.filter((service) =>
    selectedServices.some((selected) => selected.id === service.id)
  ).length;

  if (selectedCount === 0) {
    return 'none';
  }

  if (selectedCount === servicesInCluster.length) {
    return 'all';
  }

  return 'partial';
};

/**
 * Build service options from available services
 */
export const getServiceOptions = (
  services: ManagedService[]
): ServiceOption[] => {
  if (services.length === 0) {
    return [];
  }

  // Group services by cluster
  const clusterMap = new Map<string, ManagedService[]>();
  const standaloneServices: ManagedService[] = [];

  services.forEach((service) => {
    if (service.cluster) {
      if (!clusterMap.has(service.cluster)) {
        clusterMap.set(service.cluster, []);
      }
      const clusterServices = clusterMap.get(service.cluster);
      if (clusterServices) {
        clusterServices.push(service);
      }
    } else {
      standaloneServices.push(service);
    }
  });

  // Build options: standalone first, then clusters with their services
  const options: ServiceOption[] = [];

  // Add standalone services
  standaloneServices.forEach((service) => {
    options.push({
      type: 'service',
      id: service.serviceId,
      label: service.serviceName,
      serviceId: service.serviceId,
    });
  });

  // Add clusters and their services
  Array.from(clusterMap.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .forEach(([clusterName, clusterServices]) => {
      // Add cluster header as a selectable option
      options.push({
        type: 'cluster',
        id: `cluster-${clusterName}`,
        label: clusterName,
        cluster: clusterName,
      });

      // Add cluster services sorted by name
      clusterServices
        .sort((a, b) => a.serviceName.localeCompare(b.serviceName))
        .forEach((service) => {
          options.push({
            type: 'service',
            id: service.serviceId,
            label: service.serviceName,
            serviceId: service.serviceId,
            cluster: clusterName,
          });
        });
    });

  return options;
};

/**
 * Toggle all services in a cluster
 */
export const toggleClusterServices = (
  clusterName: string,
  serviceOptions: ServiceOption[],
  selectedServices: ServiceOption[]
): ServiceOption[] => {
  const servicesInCluster = serviceOptions.filter(
    (option) => option.type === 'service' && option.cluster === clusterName
  );

  const state = getClusterSelectionState(clusterName, serviceOptions, selectedServices);

  if (state === 'all') {
    // Deselect all services in this cluster
    return selectedServices.filter(
      (selected) =>
        !servicesInCluster.some((service) => service.id === selected.id)
    );
  }

  // Select all services in this cluster
  const newSelections = [...selectedServices];
  servicesInCluster.forEach((service) => {
    if (!newSelections.some((selected) => selected.id === service.id)) {
      newSelections.push(service);
    }
  });

  return newSelections;
};
