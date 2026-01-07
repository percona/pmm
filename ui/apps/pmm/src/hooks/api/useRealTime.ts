import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getRunningRealTimeAgents } from 'api/rta';
import { RunningRealTimeAgent } from 'types/rta.types';

export const useRealTimeAgents = (
  options?: Partial<UseQueryOptions<RunningRealTimeAgent[]>>
) =>
  useQuery({
    queryKey: ['rta:list-agents'],
    queryFn: () => getRunningRealTimeAgents(),
    ...options,
  });
