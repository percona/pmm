import { AdvisorCheckTriggeredBy } from 'types/advisors.types';
import { TRIGGERED_BY_LABEL } from '../insights/AdvisorInsights.utils';
import { Messages } from './AdvisorRuns.messages';

export const TRIGGERED_BY_FILTER_OPTIONS = [
  { label: Messages.filters.all, value: '' },
  ...[AdvisorCheckTriggeredBy.user, AdvisorCheckTriggeredBy.scheduler].map(
    (triggeredBy) => ({
      label: TRIGGERED_BY_LABEL[triggeredBy],
      value: triggeredBy,
    })
  ),
];
