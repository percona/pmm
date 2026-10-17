import { format } from 'date-fns';
import { AdvisorCheckTriggeredBy, Insight } from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import {
  ADVISOR_INTERVAL,
  ADVISOR_RESULT_STATUS,
  SEVERITY,
  TIME_FORMAT,
} from 'lib/constants';

export const TRIGGERED_BY_LABEL: Record<AdvisorCheckTriggeredBy, string> = {
  [AdvisorCheckTriggeredBy.user]: 'User',
  [AdvisorCheckTriggeredBy.scheduler]: 'Scheduler',
  [AdvisorCheckTriggeredBy.unspecified]: '',
};

// renders an insight as a human-readable narrative for "Copy as text"
export const insightToText = (item: Insight): string => {
  const labels = Object.entries(item.labels ?? {})
    .map(([key, value]) => `${key}=${value}`)
    .join(', ');

  const details: Array<[string, string]> = [
    ['ID', item.id],
    ['Run ID', item.runId],
    ['Check Name', item.checkName],
    ['Category', item.category],
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
    // a pending or not run check has no severity
    [
      'Severity',
      item.severity === Severity.unspecified ? '' : SEVERITY[item.severity],
    ],
    ['Read More', item.readMoreUrl],
    ['Labels', labels],
  ];

  const detailLines = details
    .filter(([, value]) => !!value)
    .map(([name, value]) => `  ${name}: ${value}`)
    .join('\n');

  const status = ADVISOR_RESULT_STATUS[item.status];
  // a pending or not run check has not completed
  const headline = item.checkedAt
    ? `The Advisor Check "${item.summary}" completed at ` +
      `${format(new Date(item.checkedAt), TIME_FORMAT)} with status "${status}".`
    : `The Advisor Check "${item.summary}" has status "${status}".`;

  return `${headline}\n\nCheck Details:\n${detailLines}`;
};
