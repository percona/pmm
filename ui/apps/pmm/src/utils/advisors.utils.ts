import { Advisor, AdvisorCheckRow, AdvisorFamily } from 'types/advisors.types';
import { ServiceType } from 'types/services.types';

export const flattenAdvisorChecks = (advisors: Advisor[]): AdvisorCheckRow[] =>
  advisors.flatMap((advisor) =>
    advisor.checks.map((check) => ({
      checkName: check.name,
      summary: check.summary,
      description: check.description,
      subcategory: advisor.subcategory,
      category: advisor.category,
      family: check.family,
      interval: check.interval,
      enabled: check.enabled,
      userDefined: check.userDefined,
      disabledServiceIds: check.disabledServiceIds ?? [],
    }))
  );

// maps a check's target DB family to the inventory service type it runs against
export const ADVISOR_FAMILY_SERVICE_TYPE: Partial<
  Record<AdvisorFamily, ServiceType>
> = {
  [AdvisorFamily.mysql]: ServiceType.mysql,
  [AdvisorFamily.postgresql]: ServiceType.posgresql,
  [AdvisorFamily.mongodb]: ServiceType.mongodb,
};
