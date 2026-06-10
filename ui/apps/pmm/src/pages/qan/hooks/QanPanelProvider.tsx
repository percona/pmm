import {
  FC,
  PropsWithChildren,
  useCallback,
  useMemo,
  useState,
} from 'react';
import { useSearchParams } from 'react-router-dom';
import type { QanGroupBy, QanPanelState } from 'types/qan.types';
import {
  appendLabelsToSearchParams,
  DEFAULT_PAGE_NUMBER,
  DEFAULT_PAGE_SIZE,
  labelsFromSearchParams,
  mergeServiceIdFromSearchParams,
  toIsoPeriod,
} from 'pages/qan/utils/qanTools';
import { parseQanColumns } from 'pages/qan/utils/qanNormalize';
import { serviceUuidFromLabels } from 'pages/qan/utils/qanServiceResolve';
import { parseQanDetailsTab } from 'pages/qan/utils/qanSectionTabs';
import { QanPanelContext } from './useQanPanelState';
import type { QanPanelActions } from './useQanPanelState';

const SPLIT_STORAGE_KEY = 'pmm-native-qan-split-ratio';

function readSplitRatio(): number {
  const raw = sessionStorage.getItem(SPLIT_STORAGE_KEY);
  const n = raw ? Number(raw) : 0.5;
  return Number.isFinite(n) && n > 0.2 && n < 0.8 ? n : 0.5;
}

function defaultTimeRange(): { from: string; to: string } {
  const to = Date.now();
  const from = to - 60 * 60 * 1000;
  return { from: toIsoPeriod(from), to: toIsoPeriod(to) };
}

export const QanPanelProvider: FC<PropsWithChildren> = ({ children }) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const [splitRatio, setSplitRatioState] = useState(readSplitRatio);

  const state = useMemo<QanPanelState>(() => {
    const fromParam = searchParams.get('from');
    const toParam = searchParams.get('to');
    const fallback = defaultTimeRange();
    const columnsRaw = searchParams.get('columns');
    const columns = parseQanColumns(columnsRaw);
    const queryId = searchParams.get('query_id') ?? searchParams.get('filter_by') ?? undefined;
    const groupBy = (searchParams.get('group_by') ?? 'queryid') as QanGroupBy;
    return {
      from: fromParam ? toIsoPeriod(Number(fromParam)) : fallback.from,
      to: toParam ? toIsoPeriod(Number(toParam)) : fallback.to,
      columns,
      labels: mergeServiceIdFromSearchParams(
        labelsFromSearchParams(searchParams),
        searchParams
      ),
      pageNumber: Number(searchParams.get('page_number') ?? DEFAULT_PAGE_NUMBER),
      pageSize: Number(searchParams.get('page_size') ?? DEFAULT_PAGE_SIZE),
      orderBy: searchParams.get('order_by') ?? `-${columns[0]}`,
      queryId,
      totals: searchParams.get('totals') === 'true',
      querySelected: !!queryId || searchParams.get('query_selected') === 'true',
      groupBy,
      openDetailsTab: parseQanDetailsTab(searchParams.get('tab') ?? searchParams.get('details_tab')),
      fingerprint: searchParams.get('fingerprint') ?? undefined,
      database: searchParams.get('selected_query_database') ?? undefined,
      dimensionSearchText: searchParams.get('dimensionSearchText') ?? '',
    };
  }, [searchParams]);

  const patchParams = useCallback(
    (patch: Record<string, string | null | undefined>) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        Object.entries(patch).forEach(([k, v]) => {
          if (v == null || v === '') next.delete(k);
          else next.set(k, v);
        });
        return next;
      }, { replace: true });
    },
    [setSearchParams]
  );

  const actions = useMemo<QanPanelActions>(
    () => ({
      setTimeRange: (from, to) => {
        patchParams({ from: String(from), to: String(to) });
      },
      setLabels: (labels) => {
        setSearchParams((prev) => {
          const next = new URLSearchParams(prev);
          appendLabelsToSearchParams(next, labels);
          return next;
        }, { replace: true });
      },
      selectQuery: (queryId, fingerprint, database, totals = false) => {
        setSearchParams((prev) => {
          const next = new URLSearchParams(prev);
          const labels = mergeServiceIdFromSearchParams(
            labelsFromSearchParams(prev),
            prev
          );
          const serviceId = serviceUuidFromLabels(labels);
          next.set('query_id', queryId);
          next.set('query_selected', 'true');
          if (totals) {
            next.set('totals', 'true');
            next.delete('tab');
            next.delete('details_tab');
          } else {
            next.delete('totals');
          }
          if (fingerprint) next.set('fingerprint', fingerprint);
          else next.delete('fingerprint');
          if (database) next.set('selected_query_database', database);
          else next.delete('selected_query_database');
          if (serviceId) {
            next.set('filter_service_id', serviceId);
            next.set('service_id', serviceId);
          }
          return next;
        }, { replace: true });
      },
      closeDetails: () => {
        patchParams({
          query_id: null,
          filter_by: null,
          query_selected: null,
          fingerprint: null,
          selected_query_database: null,
          tab: null,
          totals: null,
        });
      },
      setPage: (pageNumber, pageSize) => {
        patchParams({
          page_number: String(pageNumber),
          ...(pageSize != null ? { page_size: String(pageSize) } : {}),
        });
      },
      setOrderBy: (orderBy) => patchParams({ order_by: orderBy }),
      setGroupBy: (groupBy) => patchParams({ group_by: groupBy }),
      setColumns: (columns) => patchParams({ columns: JSON.stringify(columns) }),
      setTab: (tab) => patchParams({ tab }),
      setTotals: (totals) => patchParams({ totals: totals ? 'true' : null }),
      setSearchText: (text) => patchParams({ dimensionSearchText: text || null }),
      getSplitRatio: () => splitRatio,
      setSplitRatio: (ratio) => {
        sessionStorage.setItem(SPLIT_STORAGE_KEY, String(ratio));
        setSplitRatioState(ratio);
      },
    }),
    [patchParams, setSearchParams, splitRatio]
  );

  const value = useMemo(() => ({ state, actions }), [state, actions]);

  return (
    <QanPanelContext.Provider value={value}>{children}</QanPanelContext.Provider>
  );
};
