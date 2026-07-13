import type { QanDatabaseType, QanDetailsTab, QanPanelState } from 'types/qan.types';

export function getVisibleQanTabs(state: QanPanelState, databaseType: QanDatabaseType) {
  const groupByQuery = state.groupBy === 'queryid';
  const notTotals = !state.totals;
  return {
    details: true,
    examples: groupByQuery && notTotals,
    explainPlan:
      groupByQuery &&
      notTotals &&
      databaseType !== 'mongodb',
    tables: groupByQuery && notTotals && databaseType !== 'mongodb',
    aiInsights: groupByQuery && notTotals,
  } satisfies Record<QanDetailsTab, boolean>;
}
