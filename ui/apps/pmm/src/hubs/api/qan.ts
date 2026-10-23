import { api } from 'api/api';
import type {
  QanReportLabel,
  QanReportRequest,
  QanReportResponse,
  QanReportRow,
} from './qan.types';

export interface QanReportOptions {
  from: Date;
  to: Date;
  labels: QanReportLabel[];
  limit?: number;
}

/** `POST /v1/qan/metrics:getReport`, grouped by query id, heaviest load first. */
export const getReport = async ({
  from,
  to,
  labels,
  limit = 50,
}: QanReportOptions): Promise<QanReportRow[]> => {
  const body: QanReportRequest = {
    periodStartFrom: from.toISOString(),
    periodStartTo: to.toISOString(),
    groupBy: 'queryid',
    labels,
    columns: ['load', 'num_queries', 'query_time'],
    orderBy: '-load',
    mainMetric: 'load',
    offset: 0,
    limit,
  };
  const res = await api.post<QanReportResponse>(
    '/qan/metrics:getReport',
    body,
    // The hub shows a failed source inside its own section, so the global error toast stays quiet.
    { disableNotifications: true }
  );

  return res.data.rows ?? [];
};
