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

// A cluster spanning technologies is listed under each, and each row acts on its
// own technology's services alone.
const servicesOfCluster = (
  clusterOption: ServiceOption,
  serviceOptions: ServiceOption[]
): ServiceOption[] =>
  serviceOptions.filter(
    (option) =>
      option.type === 'service' &&
      option.cluster === clusterOption.cluster &&
      option.technology === clusterOption.technology
  );

/**
 * Get the selection state of a cluster
 */
export const getClusterSelectionState = (
  clusterOption: ServiceOption,
  serviceOptions: ServiceOption[],
  selectedServices: ServiceOption[]
): ClusterSelectionState => {
  const servicesInCluster = servicesOfCluster(clusterOption, serviceOptions);

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

  // MUI needs options sorted by group or a header repeats for every run;
  // services of unknown technology sort into one trailing group.
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

  // Across the whole list: within one group a mixed cluster looks uniform, and its header
  // would become selectable.
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

// Undefined for a cluster whose services disagree on technology.
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

const clusterNameOf = (service: AvailableService | RealtimeSession): string =>
  'cluster' in service ? service.cluster : service.clusterName;

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
      technology: technologyLabel,
    });
  });

  // Add clusters and their services
  Array.from(clusterMap.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .forEach(([clusterName, clusterServices]) => {
      // Add cluster header as a selectable option
      options.push({
        type: 'cluster',
        // A cluster spanning technologies appears under each, so the id includes it.
        id: `cluster-${technologyLabel}-${clusterName}`,
        label: clusterName,
        cluster: clusterName,
        serviceType: clusterTechnologies.get(clusterName),
        technology: technologyLabel,
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
            technology: technologyLabel,
          });
        });
    });

  return options;
};

/**
 * Toggle all services in a cluster
 */
export const toggleClusterServices = (
  clusterOption: ServiceOption,
  serviceOptions: ServiceOption[],
  selectedServices: ServiceOption[]
): ServiceOption[] => {
  const servicesInCluster = servicesOfCluster(clusterOption, serviceOptions);

  const state = getClusterSelectionState(
    clusterOption,
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
 * Whether an option is off-limits in a single-technology view: a mixed-technology
 * cluster always is, anything else once another technology has been picked.
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
