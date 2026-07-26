import {
  Advisor,
  AdvisorCheckRow,
  AdvisorTechnology,
} from 'types/advisors.types';
import { ServiceType } from 'types/services.types';

export const flattenAdvisorChecks = (advisors: Advisor[]): AdvisorCheckRow[] =>
  advisors.flatMap((advisor) =>
    advisor.checks.map((check) => ({
      checkName: check.name,
      summary: check.summary,
      description: check.description,
      subcategory: advisor.subcategory,
      category: advisor.category,
      technology: check.technology,
      interval: check.interval,
      enabled: check.enabled,
      userDefined: check.userDefined,
      disabledServiceIds: check.disabledServiceIds ?? [],
    }))
  );

// maps a check's target DB technology to the inventory service type it runs against
export const ADVISOR_TECHNOLOGY_SERVICE_TYPE: Partial<
  Record<AdvisorTechnology, ServiceType>
> = {
  [AdvisorTechnology.mysql]: ServiceType.mysql,
  [AdvisorTechnology.postgresql]: ServiceType.posgresql,
  [AdvisorTechnology.mongodb]: ServiceType.mongodb,
};
