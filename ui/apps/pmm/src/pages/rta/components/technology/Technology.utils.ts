import { ServiceType } from 'types/services.types';
import { Messages } from './Technology.messages';

// Unsupported technologies get no label rather than a raw enum value.
const TECHNOLOGY_LABELS: Partial<Record<ServiceType, string>> = {
  [ServiceType.mongodb]: Messages.mongodb,
  [ServiceType.mysql]: Messages.mysql,
  [ServiceType.posgresql]: Messages.postgresql,
};

export const technologyLabel = (serviceType?: ServiceType): string =>
  (serviceType && TECHNOLOGY_LABELS[serviceType]) || '';

// Undefined when the services disagree on technology.
export const sharedTechnology = (
  serviceTypes: (ServiceType | undefined)[]
): ServiceType | undefined => {
  const distinct = new Set(serviceTypes);

  return distinct.size === 1 ? serviceTypes[0] : undefined;
};
