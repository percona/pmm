import { ADVISOR_FAMILY, ADVISOR_INTERVAL } from 'lib/constants';
import {
  Advisor,
  AdvisorCheckRow,
  CategorizedAdvisor,
} from 'types/advisors.types';

export const flattenAdvisorChecks = (advisors: Advisor[]): AdvisorCheckRow[] =>
  advisors.flatMap((advisor) =>
    advisor.checks.map((check) => ({
      checkName: check.name,
      summary: check.summary,
      description: check.description,
      advisorName: advisor.summary,
      category: advisor.category,
      family: check.family,
      interval: check.interval,
      enabled: check.enabled,
    }))
  );

export const groupAdvisorsIntoCategories = (
  advisors: Advisor[]
): CategorizedAdvisor => {
  const result: CategorizedAdvisor = {};

  advisors.forEach((advisor) => {
    const { category, summary, checks } = advisor;

    const modifiedChecks = checks.map((check) => ({
      ...check,
      familyName: check.family ? ADVISOR_FAMILY[check.family] : undefined,
      intervalName: check.interval
        ? ADVISOR_INTERVAL[check.interval]
        : undefined,
    }));

    if (!result[category]) {
      result[category] = {};
    }

    if (!result[category][summary]) {
      result[category][summary] = { ...advisor, checks: [...modifiedChecks] };
    }
  });
  return result;
};
