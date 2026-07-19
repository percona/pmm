import { Advisor, AdvisorCheckRow } from 'types/advisors.types';

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
    }))
  );
