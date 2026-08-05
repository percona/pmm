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

// hasMixedTechnologies reports whether a set of services spans more than one
// technology. The technology is only worth showing in that case: on a
// single-engine install it is the same answer on every row.
// Only named technologies count, so a service we cannot label - the server sends
// SERVICE_TYPE_UNSPECIFIED rather than omitting the field - cannot switch the
// display on and then render an empty cell.
export const hasMixedTechnologies = (
  serviceTypes: (ServiceType | undefined)[]
): boolean => new Set(serviceTypes.filter(technologyLabel)).size > 1;

// sharedTechnology returns the technology common to every service, or undefined
// when they disagree - used for rows that stand for a whole cluster.
export const sharedTechnology = (
  serviceTypes: (ServiceType | undefined)[]
): ServiceType | undefined => {
  const distinct = new Set(serviceTypes);

  return distinct.size === 1 ? serviceTypes[0] : undefined;
};
