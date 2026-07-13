import { createContext, useContext } from 'react';
import type {
  QanDetailsTab,
  QanGroupBy,
  QanLabelsMap,
  QanPanelState,
} from 'types/qan.types';

export interface QanPanelActions {
  setTimeRange: (from: number, to: number) => void;
  setLabels: (labels: QanLabelsMap) => void;
  selectQuery: (queryId: string, fingerprint?: string, database?: string, totals?: boolean) => void;
  closeDetails: () => void;
  setPage: (pageNumber: number, pageSize?: number) => void;
  setOrderBy: (orderBy: string) => void;
  setGroupBy: (groupBy: QanGroupBy) => void;
  setColumns: (columns: string[]) => void;
  setTab: (tab: QanDetailsTab) => void;
  setTotals: (totals: boolean) => void;
  setSearchText: (text: string) => void;
  getSplitRatio: () => number;
  setSplitRatio: (ratio: number) => void;
}

export type QanContextValue = {
  state: QanPanelState;
  actions: QanPanelActions;
};

export const QanPanelContext = createContext<QanContextValue | null>(null);

export function useQanPanel(): QanContextValue {
  const ctx = useContext(QanPanelContext);
  if (!ctx) throw new Error('useQanPanel must be used within QanPanelProvider');
  return ctx;
}

export function useQanPanelState(): QanPanelState {
  return useQanPanel().state;
}

export function useQanPanelActions(): QanPanelActions {
  return useQanPanel().actions;
}
