import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getRecentQANData, getQANMetricsNames, getQANFilters, QANReportResponse, QANMetricsNamesResponse, QANFiltersResponse, QANFiltersRequest, QANLabel } from 'api/qan';

export const useRecentQANData = (
  hoursBack: number = 24,
  limit: number = 10,
  filters?: QANLabel[],
  options?: Partial<UseQueryOptions<QANReportResponse>>
) =>
  useQuery({
    queryKey: ['qan', 'recent', hoursBack, limit, filters],
    queryFn: () => getRecentQANData(hoursBack, limit, filters),
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 1, // Only retry once since QAN data might not be available in dev
    ...options,
  });

export const useQANMetricsNames = (
  options?: Partial<UseQueryOptions<QANMetricsNamesResponse>>
) =>
  useQuery({
    queryKey: ['qan', 'metricsNames'],
    queryFn: getQANMetricsNames,
    staleTime: 30 * 60 * 1000, // 30 minutes - metrics names don't change often
    retry: 1,
    ...options,
  });

export const useQANFilters = (
  request: QANFiltersRequest,
  options?: Partial<UseQueryOptions<QANFiltersResponse>>
) =>
  useQuery({
    queryKey: ['qan', 'filters', request],
    queryFn: () => getQANFilters(request),
    staleTime: 10 * 60 * 1000, // 10 minutes
    retry: 1,
    ...options,
  }); 