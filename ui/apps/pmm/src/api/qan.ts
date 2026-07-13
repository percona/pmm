import { api } from './api';
import type {
  QanExplainResponse,
  QanFiltersResponse,
  QanGetExampleResponse,
  QanGetHistogramResponse,
  QanGetMetricNamesResponse,
  QanGetMetricsRequest,
  QanGetMetricsResponse,
  QanGetReportRequest,
  QanGetReportResponse,
  QanGetSchemaResponse,
  QanLabelFilter,
} from 'types/qan.types';

export const getQanReport = async (
  body: QanGetReportRequest
): Promise<QanGetReportResponse> => {
  const res = await api.post<QanGetReportResponse>('/qan/metrics:getReport', body);
  return res.data;
};

export const getQanFilters = async (body: {
  labels: QanLabelFilter[];
  mainMetricName: string;
  periodStartFrom: string;
  periodStartTo: string;
}): Promise<QanFiltersResponse> => {
  const res = await api.post<QanFiltersResponse>('/qan/metrics:getFilters', body);
  return res.data;
};

export const getQanMetricNames = async (body: {
  mainMetricName: string;
  groupBy: string;
}): Promise<QanGetMetricNamesResponse> => {
  const res = await api.post<QanGetMetricNamesResponse>('/qan/metrics:getNames', body);
  return res.data;
};

export const getQanMetrics = async (
  body: QanGetMetricsRequest
): Promise<QanGetMetricsResponse> => {
  const res = await api.post<QanGetMetricsResponse>('/qan:getMetrics', body);
  return res.data;
};

export const getQanHistogram = async (body: {
  queryid: string;
  labels: QanLabelFilter[];
  periodStartFrom: string;
  periodStartTo: string;
}): Promise<QanGetHistogramResponse> => {
  const res = await api.post<QanGetHistogramResponse>('/qan:getHistogram', body);
  return res.data;
};

export const getQanQueryExample = async (body: {
  filterBy: string;
  groupBy: string;
  labels: QanLabelFilter[];
  periodStartFrom: string;
  periodStartTo: string;
}): Promise<QanGetExampleResponse> => {
  const res = await api.post<QanGetExampleResponse>('/qan/query:getExample', body);
  return res.data;
};

export const explainQanFingerprint = async (body: {
  queryId: string;
  serviceId: string;
  database?: string;
  example?: string;
  placeholders?: Record<string, string>;
}): Promise<QanExplainResponse> => {
  const res = await api.post<QanExplainResponse>('/qan:explainFingerprint', body);
  return res.data;
};

export const getQanQuerySchema = async (body: {
  filterBy: string;
  groupBy: string;
  labels: QanLabelFilter[];
  periodStartFrom: string;
  periodStartTo: string;
  tables?: string[];
}): Promise<QanGetSchemaResponse> => {
  const res = await api.post<QanGetSchemaResponse>('/qan/query:getSchema', body);
  return res.data;
};

export const getQanQueryPlan = async (queryId: string): Promise<{ plan?: string }> => {
  const res = await api.get<{ plan?: string }>(`/qan/query/${encodeURIComponent(queryId)}/plan`);
  return res.data;
};

export const qanQueryExists = async (body: {
  filterBy: string;
  groupBy: string;
  labels: QanLabelFilter[];
  periodStartFrom: string;
  periodStartTo: string;
}): Promise<{ exists?: boolean }> => {
  const res = await api.post<{ exists?: boolean }>('/qan/query:exists', body);
  return res.data;
};
