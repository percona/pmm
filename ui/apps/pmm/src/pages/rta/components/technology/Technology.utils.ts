import { ServiceType } from 'types/services.types';
import { Messages } from './Technology.messages';

// Only the technologies Real-Time Analytics supports are named here; anything
// else renders without a label rather than as an empty or raw enum value.
const TECHNOLOGY_LABELS: Partial<Record<ServiceType, string>> = {
  [ServiceType.mongodb]: Messages.mongodb,
  [ServiceType.mysql]: Messages.mysql,
};

export const technologyLabel = (serviceType?: ServiceType): string =>
  (serviceType && TECHNOLOGY_LABELS[serviceType]) || '';

// sharedTechnology returns the technology common to every service, or undefined
// when they disagree - used for rows that stand for a whole cluster, and to
// decide which technology a selection of services belongs to.
export const sharedTechnology = (
  serviceTypes: (ServiceType | undefined)[]
): ServiceType | undefined => {
  const distinct = new Set(serviceTypes);

  return distinct.size === 1 ? serviceTypes[0] : undefined;
};
