import { UseQueryOptions, useQuery } from '@tanstack/react-query';
import { getReadiness } from 'api/ready';

export const useReadiness = (options?: Partial<UseQueryOptions>) =>
  useQuery({
    queryKey: ['readiness'],
    queryFn: async () => getReadiness(),
    ...options,
  });

export const useWaitForReadiness = () => {
  const { refetch } = useReadiness({
    refetchOnMount: false,
    retry: true,
    retryDelay: 5000,
  });

  return { waitForReadiness: refetch };
};
