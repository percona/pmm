import type { QanDatabaseType, QanDetailsTab, QanPanelState } from 'types/qan.types';

export function getVisibleQanTabs(state: QanPanelState, databaseType: QanDatabaseType) {
  const groupByQuery = state.groupBy === 'queryid';
  const notTotals = !state.totals;
  return {
    details: true,
    examples: groupByQuery && notTotals,
    explain: groupByQuery && notTotals && databaseType !== 'mongodb' && databaseType !== 'postgresql',
    tables: groupByQuery && notTotals && databaseType !== 'mongodb',
    plan: groupByQuery && notTotals && databaseType === 'postgresql',
    aiInsights: groupByQuery && notTotals,
  } satisfies Record<QanDetailsTab, boolean>;
}
