import { format } from 'date-fns';
import {
  AdvisorCheckTriggeredBy,
  CheckResultHistoryItem,
} from 'types/advisors.types';
import {
  ADVISOR_INTERVAL,
  ADVISOR_RESULT_STATUS,
  SEVERITY,
  TIME_FORMAT,
} from 'lib/constants';
import { capitalize } from 'utils/text.utils';

const TRIGGERED_BY_LABEL: Record<AdvisorCheckTriggeredBy, string> = {
  [AdvisorCheckTriggeredBy.user]: 'User',
  [AdvisorCheckTriggeredBy.scheduler]: 'Scheduler',
  [AdvisorCheckTriggeredBy.unspecified]: '',
};

// renders an insight as a human-readable narrative for "Copy as text"
export const insightToText = (item: CheckResultHistoryItem): string => {
  const checkedAt = item.checkedAt
    ? format(new Date(item.checkedAt), TIME_FORMAT)
    : '';
  const labels = Object.entries(item.labels ?? {})
    .map(([key, value]) => `${key}=${value}`)
    .join(', ');

  const details: Array<[string, string]> = [
    ['ID', item.id],
    ['Check Run ID', item.runId],
    ['Check Name', item.checkName],
    ['Advisor', item.advisorName],
    ['Category', capitalize(item.category)],
    ['Service Name', item.serviceName],
    ['Service Type', item.serviceType],
    ['Node Name', item.nodeName],
    ['Environment', item.environment],
    ['Cluster', item.cluster],
    ['Replication Set', item.replicationSet],
    ['Interval', ADVISOR_INTERVAL[item.interval]],
    ['Triggered By', TRIGGERED_BY_LABEL[item.triggeredBy]],
    ['Read', item.isRead ? 'Read' : 'Unread'],
    ['Summary', item.summary],
    ['Description', item.description],
    ['Outcome', item.outcome],
    ['Severity', SEVERITY[item.severity]],
    ['Read More', item.readMoreUrl],
    ['Labels', labels],
  ];

  const detailLines = details
    .filter(([, value]) => !!value)
    .map(([name, value]) => `  ${name}: ${value}`)
    .join('\n');

  return (
    `The Advisor Check "${item.summary}" completed at ${checkedAt} ` +
    `with status "${ADVISOR_RESULT_STATUS[item.status]}".\n` +
    `\n` +
    `Check Details:\n` +
    detailLines
  );
};
