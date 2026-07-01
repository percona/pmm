import { getVisibleQanTabs } from './qanTabVisibility';
import type { QanPanelState } from 'types/qan.types';

const baseState: QanPanelState = {
  from: '',
  to: '',
  columns: ['load'],
  labels: {},
  pageNumber: 1,
  pageSize: 25,
  orderBy: '-load',
  totals: false,
  querySelected: true,
  groupBy: 'queryid',
  openDetailsTab: 'details',
};

describe('getVisibleQanTabs', () => {
  it('shows core tabs for mysql queryid grouping', () => {
    const v = getVisibleQanTabs(baseState, 'mysql');
    expect(v.details).toBe(true);
    expect(v.examples).toBe(true);
    expect(v.explainPlan).toBe(true);
    expect(v.aiInsights).toBe(true);
  });

  it('shows explain plan for postgresql', () => {
    const v = getVisibleQanTabs(baseState, 'postgresql');
    expect(v.explainPlan).toBe(true);
  });

  it('hides explain plan and tables for mongodb', () => {
    const v = getVisibleQanTabs(baseState, 'mongodb');
    expect(v.explainPlan).toBe(false);
    expect(v.tables).toBe(false);
  });
});
