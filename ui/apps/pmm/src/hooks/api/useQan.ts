import { keepPreviousData, useQuery } from '@tanstack/react-query';
import {
  explainQanFingerprint,
  getQanFilters,
  getQanMetricNames,
  getQanMetrics,
  getQanQueryExample,
  getQanQueryPlan,
  getQanQuerySchema,
  getQanReport,
} from 'api/qan';
import type {
  QanGetMetricsRequest,
  QanGetReportRequest,
  QanLabelFilter,
} from 'types/qan.types';

export const QAN_KEYS = {
  report: (params: QanGetReportRequest) => ['qan', 'report', params] as const,
  filters: (params: {
    labels: QanLabelFilter[];
    mainMetricName: string;
    periodStartFrom: string;
    periodStartTo: string;
  }) => ['qan', 'filters', params] as const,
  metricNames: (mainMetric: string, groupBy: string) =>
    ['qan', 'metricNames', mainMetric, groupBy] as const,
  metrics: (params: QanGetMetricsRequest) => ['qan', 'metrics', params] as const,
  examples: (params: {
    filterBy: string;
    groupBy: string;
    labels: QanLabelFilter[];
    periodStartFrom: string;
    periodStartTo: string;
  }) => ['qan', 'examples', params] as const,
  explain: (params: {
    queryId: string;
    serviceId: string;
    database?: string;
    example?: string;
  }) => ['qan', 'explain', params] as const,
  schema: (params: {
    filterBy: string;
    groupBy: string;
    labels: QanLabelFilter[];
    periodStartFrom: string;
    periodStartTo: string;
    tables?: string[];
  }) => ['qan', 'schema', params] as const,
  plan: (queryId: string) => ['qan', 'plan', queryId] as const,
};

export function useQanReport(params: QanGetReportRequest, enabled = true) {
  return useQuery({
    queryKey: QAN_KEYS.report(params),
    queryFn: () => getQanReport(params),
    enabled,
    placeholderData: keepPreviousData,
  });
}

export function useQanFilters(
  params: {
    labels: QanLabelFilter[];
    mainMetricName: string;
    periodStartFrom: string;
    periodStartTo: string;
  },
  enabled = true
) {
  return useQuery({
    queryKey: QAN_KEYS.filters(params),
    queryFn: () => getQanFilters(params),
    enabled,
  });
}

export function useQanMetricNames(mainMetric: string, groupBy: string, enabled = true) {
  return useQuery({
    queryKey: QAN_KEYS.metricNames(mainMetric, groupBy),
    queryFn: () => getQanMetricNames({ mainMetricName: mainMetric, groupBy }),
    enabled: enabled && !!mainMetric,
  });
}

export function useQanMetrics(params: QanGetMetricsRequest, enabled = true) {
  return useQuery({
    queryKey: QAN_KEYS.metrics(params),
    queryFn: () => getQanMetrics(params),
    enabled: enabled && !!params.filterBy,
  });
}

export function useQanExamples(
  params: {
    filterBy: string;
    groupBy: string;
    labels: QanLabelFilter[];
    periodStartFrom: string;
    periodStartTo: string;
  },
  enabled = true
) {
  return useQuery({
    queryKey: QAN_KEYS.examples(params),
    queryFn: () => getQanQueryExample(params),
    enabled: enabled && !!params.filterBy,
  });
}

export function useQanExplain(
  params: {
    queryId: string;
    serviceId: string;
    database?: string;
    example?: string;
    placeholders?: Record<string, string>;
  },
  enabled = true
) {
  return useQuery({
    queryKey: QAN_KEYS.explain(params),
    queryFn: () => explainQanFingerprint(params),
    enabled: enabled && !!params.queryId && !!params.serviceId,
  });
}

export function useQanSchema(
  params: {
    filterBy: string;
    groupBy: string;
    labels: QanLabelFilter[];
    periodStartFrom: string;
    periodStartTo: string;
    tables?: string[];
  },
  enabled = true
) {
  return useQuery({
    queryKey: QAN_KEYS.schema(params),
    queryFn: () => getQanQuerySchema(params),
    enabled: enabled && !!params.filterBy,
  });
}

export function useQanPlan(queryId: string, enabled = true) {
  return useQuery({
    queryKey: QAN_KEYS.plan(queryId),
    queryFn: () => getQanQueryPlan(queryId),
    enabled: enabled && !!queryId,
  });
}
