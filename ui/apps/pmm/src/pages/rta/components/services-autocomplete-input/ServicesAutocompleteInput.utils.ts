import {
  ClusterSelectionState,
  ServiceOption,
} from './ServicesAutocompleteInput.types';
import { AvailableService, RealtimeSession } from 'types/rta.types';
import { ServiceType } from 'types/services.types';
import {
  sharedTechnology,
  technologyLabel,
} from 'pages/rta/components/technology';

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
  services: AvailableService[] | RealtimeSession[]
): ServiceOption[] => {
  if (services.length === 0) {
    return [];
  }

  // The picker groups options by technology and MUI expects the list to arrive
  // already sorted by group, otherwise a header repeats for every run of
  // options. Services we cannot name sort last, so they end up in one trailing
  // group rather than scattered.
  const byTechnology = new Map<
    string,
    (AvailableService | RealtimeSession)[]
  >();

  services.forEach((service) => {
    const label = technologyLabel(service.serviceType);
    const group = byTechnology.get(label);

    if (group) {
      group.push(service);
    } else {
      byTechnology.set(label, [service]);
    }
  });

  // Worked out across the whole list, before the split below. A cluster spanning technologies
  // has services in more than one group, and each group on its own looks uniform -- so asking
  // the group would report a single technology for a cluster that has none, and the header
  // would become selectable. Selecting it seeds a mixed set in one click, which is exactly
  // what isServiceOptionDisabled exists to prevent.
  const clusterTechnologies = getClusterTechnologies(services);

  return Array.from(byTechnology.keys())
    .sort((a, b) => {
      if (!a || !b) {
        return a ? -1 : 1;
      }

      return a.localeCompare(b);
    })
    .flatMap((label) =>
      getClusterOptions(
        byTechnology.get(label) ?? [],
        clusterTechnologies,
        label
      )
    );
};

// getClusterTechnologies maps each cluster name to the technology all of its services share,
// or undefined when they disagree.
const getClusterTechnologies = (
  services: (AvailableService | RealtimeSession)[]
): Map<string, ServiceType | undefined> => {
  const byCluster = new Map<string, (ServiceType | undefined)[]>();

  services.forEach((service) => {
    const clusterName = clusterNameOf(service);

    if (!clusterName) {
      return;
    }

    const seen = byCluster.get(clusterName);

    if (seen) {
      seen.push(service.serviceType);
    } else {
      byCluster.set(clusterName, [service.serviceType]);
    }
  });

  return new Map(
    Array.from(byCluster.entries()).map(([clusterName, types]) => [
      clusterName,
      sharedTechnology(types),
    ])
  );
};

// clusterNameOf reads the cluster from either shape the picker is fed.
const clusterNameOf = (service: AvailableService | RealtimeSession): string =>
  'cluster' in service ? service.cluster : service.clusterName;

// getClusterOptions lays out one technology's services: standalone ones first,
// then each cluster header followed by its services.
const getClusterOptions = (
  services: (AvailableService | RealtimeSession)[],
  clusterTechnologies: Map<string, ServiceType | undefined>,
  technologyLabel: string
): ServiceOption[] => {
  // Group services by cluster
  const clusterMap = new Map<string, (AvailableService | RealtimeSession)[]>();
  const standaloneServices: (AvailableService | RealtimeSession)[] = [];

  services.forEach((service) => {
    let clusterName = '';

    if ('cluster' in service) {
      clusterName = service.cluster;
    } else {
      clusterName = service.clusterName;
    }

    if (clusterName) {
      if (!clusterMap.has(clusterName)) {
        clusterMap.set(clusterName, []);
      }
      const clusterServices = clusterMap.get(clusterName);
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
      serviceType: service.serviceType,
    });
  });

  // Add clusters and their services
  Array.from(clusterMap.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .forEach(([clusterName, clusterServices]) => {
      // Add cluster header as a selectable option
      options.push({
        type: 'cluster',
        // A cluster spanning technologies appears once under each of them, so the technology
        // is part of the identity: two options sharing an id would collide as React keys.
        id: `cluster-${technologyLabel}-${clusterName}`,
        label: clusterName,
        cluster: clusterName,
        serviceType: clusterTechnologies.get(clusterName),
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
            serviceType: service.serviceType,
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

  const state = getClusterSelectionState(
    clusterName,
    serviceOptions,
    selectedServices
  );

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

export const getServiceIds = (serviceOptions: ServiceOption[]): string[] =>
  serviceOptions
    .filter((option) => option.type === 'service' && option.serviceId)
    .map((option) => option.serviceId!);

/**
 * Whether an option may not be picked while one view of live queries is being
 * assembled. The first pick fixes the technology; a cluster whose services
 * disagree carries none of its own, and selecting it would seed a mixed set in
 * one click, so it stays disabled even before a technology has been fixed.
 */
export const isServiceOptionDisabled = (
  option: ServiceOption,
  selectedTechnology: ServiceType | undefined,
  singleTechnology: boolean
): boolean =>
  singleTechnology &&
  (option.serviceType === undefined ||
    (selectedTechnology !== undefined &&
      option.serviceType !== selectedTechnology));
