import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getRealTimeData,
  getRealTimeServices,
  enableRealTimeAnalytics,
  disableRealTimeAnalytics,
  updateRealTimeConfig,
} from 'api/realtime';
import { RealTimeConfig } from 'types/realtime.types';

export const useRealTimeData = (serviceId?: string, refetchInterval = 2000) =>
  useQuery({
    queryKey: ['realtime/data', serviceId],
    queryFn: () => getRealTimeData(serviceId),
    refetchInterval,
    refetchIntervalInBackground: true,
    staleTime: 0, // Always consider data stale for real-time updates
    gcTime: 0, // Don't cache data to prevent accumulation
  });

export const useRealTimeServices = () =>
  useQuery({
    queryKey: ['realtime/services'],
    queryFn: getRealTimeServices,
  });

export const useEnableRealTimeAnalytics = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: enableRealTimeAnalytics,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['realtime/services'] });
    },
  });
};

export const useDisableRealTimeAnalytics = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: disableRealTimeAnalytics,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['realtime/services'] });
    },
  });
};

export const useUpdateRealTimeConfig = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ serviceId, config }: { serviceId: string; config: RealTimeConfig }) =>
      updateRealTimeConfig(serviceId, config),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['realtime/services'] });
    },
  });
};
