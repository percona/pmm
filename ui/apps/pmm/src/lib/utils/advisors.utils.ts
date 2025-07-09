import { ADVISOR_FAMILY, ADVISOR_INTERVAL } from 'lib/constants';
import { Advisor, CategorizedAdvisor } from 'types/advisors.types';

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
