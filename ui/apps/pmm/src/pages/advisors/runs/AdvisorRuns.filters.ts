import { AdvisorCheckTriggeredBy } from 'types/advisors.types';
import { TRIGGERED_BY_LABEL } from '../insights/AdvisorInsights.utils';

// query-string params remembered across visits
export const RUN_FILTER_PARAMS = ['triggeredBy', 'pageSize'];

export const TRIGGERED_BY_FILTER_OPTIONS = [
  AdvisorCheckTriggeredBy.user,
  AdvisorCheckTriggeredBy.scheduler,
].map((triggeredBy) => ({
  label: TRIGGERED_BY_LABEL[triggeredBy],
  value: triggeredBy,
}));
