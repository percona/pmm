import { format } from 'date-fns/format';
import { type MRT_Row } from 'material-react-table';
import { ALL_STATES_FILTER, SILENCED_FILTER } from '../AlertsPage.constants';
import { AlertRow, AlertsTableRow } from '../AlertsPage.types';
import { groupAlertsByNode } from '../AlertsPage.utils';
import { GRAFANA_SUB_PATH, TIME_FORMAT } from 'lib/constants';
import { tz } from '@date-fns/tz/tz';
import {
  createRelativeUrl,
  makeLabelBasedSilenceLink,
} from 'utils/alerting.utils';

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
  let result = rows;

  if (selectedState === SILENCED_FILTER) {
    result = rows.filter((row) => row.silenced);
  } else if (selectedState !== ALL_STATES_FILTER) {
    result = rows.filter((row) => row.state === selectedState && !row.silenced);
  }

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

// Link to Grafana's silences list, where a user can expire (unsilence) the
// silence suppressing an alert — matching the fired-alerts page behaviour.
export const createUnsilenceUrl = () => createRelativeUrl('/alerting/silences');

const STATE_ORDER = {
  Normal: 0,
  Pending: 1,
  Alerting: 2,
  Recovering: 3,
  NoData: 4,
  Error: 5,
  Silenced: 6,
};

const NO_VALUE = '[no value]';

export const sortByState = (a: AlertsTableRow, b: AlertsTableRow) => {
  if (a.type === 'node' || b.type === 'node') {
    return 0;
  }
  const stateA = a.silenced ? 'Silenced' : a.state;
  const stateB = b.silenced ? 'Silenced' : b.state;
  return (STATE_ORDER[stateA] ?? 0) - (STATE_ORDER[stateB] ?? 0);
};

export const sortByNode = (a: AlertsTableRow, b: AlertsTableRow) => {
  if (a.type === 'node' || b.type === 'node') {
    return 0;
  }

  const valueA = a.nodeId === NO_VALUE ? '' : a.nodeId;
  const valueB = b.nodeId === NO_VALUE ? '' : b.nodeId;
  return valueA.localeCompare(valueB, undefined, { sensitivity: 'base' });
};

export const sortByService = (a: AlertsTableRow, b: AlertsTableRow) => {
  if (a.type === 'node' || b.type === 'node') {
    return 0;
  }

  const valueA = a.serviceName === NO_VALUE ? '' : a.serviceName;
  const valueB = b.serviceName === NO_VALUE ? '' : b.serviceName;
  return valueA.localeCompare(valueB, undefined, { sensitivity: 'base' });
};
