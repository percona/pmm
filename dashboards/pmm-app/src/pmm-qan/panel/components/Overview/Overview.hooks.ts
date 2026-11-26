import {
  useContext, useEffect, useMemo, useState,
} from 'react';
import axios from 'axios';
import { QueryAnalyticsProvider } from 'pmm-qan/panel/provider/provider';
import OverviewService from './Overview.service';
import { DataInterface } from './Overview.types';
import { getOverviewColumn } from './components/MetricColumns/MetricColumns';
import { getDefaultColumns } from './components/DefaultColumns/DefaultColumns';

export const useOverviewTable = (setTotal): [DataInterface, boolean] => {
  const {
    panelState: {
      labels, columns, pageNumber, pageSize, orderBy, from, to, groupBy, dimensionSearchText,
    },
  } = useContext(QueryAnalyticsProvider);
  const [data, setData] = useState<DataInterface>({ rows: [], columns: [] });
  const [loading, setLoading] = useState(false);
  const defaultColumns = useMemo(() => getDefaultColumns(), []);

  useEffect(() => {
    // Cancel this request once a newer one supersedes it, see PMM-15883.
    const source = axios.CancelToken.source();

    const updateInstances = async () => {
      try {
        setLoading(true);
        const result = await OverviewService.getReport({
          labels,
          columns,
          pageNumber,
          pageSize,
          orderBy,
          from,
          to,
          groupBy,
          dimensionSearchText,
        }, source.token);

        setTotal(result.total_rows);

        const mainMetric = columns[0];
        // eslint-disable-next-line max-len
        const metricsColumns = columns.map((key, index) => getOverviewColumn(key, index, result.rows[0], orderBy, mainMetric));

        const allColumns = [...defaultColumns, ...metricsColumns];

        setData({ rows: result.rows, columns: allColumns });
        setLoading(false);
      } catch (e) {
        if (!axios.isCancel(e)) {
          setLoading(false);
        }
      }
    };

    updateInstances().then(() => {});

    return () => {
      source.cancel();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [columns, pageNumber, pageSize, groupBy, labels, orderBy, from, to, dimensionSearchText]);

  return [data, loading];
};
