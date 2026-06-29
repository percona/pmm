import { format } from 'date-fns/format';
import { ALL_STATES_FILTER } from '../AlertsPage.constants';
import { AlertRow, AlertsTableRow } from '../AlertsPage.types';
import { groupAlertsByNode } from '../AlertsPage.utils';
import { GRAFANA_SUB_PATH, TIME_FORMAT } from 'lib/constants';
import { tz } from '@date-fns/tz/tz';
import { makeLabelBasedSilenceLink } from 'utils/alerting.utils';

export type GetFilteredDataParams = {
  rows: AlertRow[];
  groupByNodes: boolean;
  selectedState: string;
  timezone?: string;
};

export const getTableRows = ({
  rows,
  groupByNodes,
  selectedState,
  timezone,
}: GetFilteredDataParams): AlertsTableRow[] => {
  let result = rows.map((row) => ({ ...row, timezone }));

  if (selectedState !== ALL_STATES_FILTER) {
    result = result.filter((r) => r.state === selectedState);
  }

  return groupByNodes ? groupAlertsByNode(result) : result;
};

export const formatTriggeredAt = (
  triggeredAt: string | undefined,
  timezone?: string
) => {
  if (!triggeredAt) {
    return null;
  }

  const date = new Date(triggeredAt);

  if (Number.isNaN(date.getTime())) {
    return '-';
  }

  return format(date, TIME_FORMAT, { in: timezone ? tz(timezone) : undefined });
};

export const createAlertRuleViewUrl = (ruleGroupUid: string) =>
  `${GRAFANA_SUB_PATH}/alerting/grafana/${ruleGroupUid}/view`;

export const createAlertRuleEditUrl = (ruleGroupUid: string) =>
  `${GRAFANA_SUB_PATH}/alerting/${ruleGroupUid}/edit`;

export const createSilenceUrl = (labels: Record<string, string>) =>
  makeLabelBasedSilenceLink('grafana', labels);
