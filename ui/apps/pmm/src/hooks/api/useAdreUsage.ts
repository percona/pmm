import { useQuery } from '@tanstack/react-query';
import { getAdreUsageEvents, getAdreUsageSummary, type AdreUsageSummaryResponse } from 'api/adre';
import { getInvestigationUsage } from 'api/investigations';

export function useAdreUsageSummary(params?: {
  from?: string;
  to?: string;
  groupBy?: string;
  feature?: string;
  model?: string;
}) {
  return useQuery<AdreUsageSummaryResponse>({
    queryKey: ['adreUsageSummary', params],
    queryFn: () => getAdreUsageSummary(params),
  });
}

export function useAdreUsageEvents(params?: {
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
  feature?: string;
  model?: string;
  investigationId?: string;
}) {
  return useQuery({
    queryKey: ['adreUsageEvents', params],
    queryFn: () => getAdreUsageEvents(params),
  });
}

export function useInvestigationUsage(
  investigationId: string | undefined,
  options?: { refetchInterval?: number | false }
) {
  return useQuery({
    queryKey: ['investigationUsage', investigationId],
    queryFn: () => getInvestigationUsage(investigationId!),
    enabled: !!investigationId,
    refetchInterval: options?.refetchInterval,
  });
}
