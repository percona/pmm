import { format } from 'date-fns/format';
import { type MRT_Row } from 'material-react-table';
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
};

export const getTableRows = ({
  rows,
  groupByNodes,
  selectedState,
}: GetFilteredDataParams): AlertsTableRow[] => {
  const result =
    selectedState === ALL_STATES_FILTER
      ? rows
      : rows.filter((row) => row.state === selectedState);

  return groupByNodes ? groupAlertsByNode(result) : result;
};

const toTimestamp = (bound: unknown): number | undefined => {
  if (!(bound instanceof Date)) {
    return undefined;
  }

  const timestamp = bound.getTime();
  return Number.isNaN(timestamp) ? undefined : timestamp;
};

export const filterTriggeredAt = (
  row: MRT_Row<AlertsTableRow>,
  id: string,
  filterValue: [unknown, unknown]
) => {
  const from = toTimestamp(filterValue[0]);
  const to = toTimestamp(filterValue[1]);

  if (from === undefined && to === undefined) {
    return true;
  }

  const value = toTimestamp(row.getValue<Date | undefined>(id));

  if (value === undefined) {
    return false;
  }

  return (
    (from === undefined || value >= from) && (to === undefined || value <= to)
  );
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
