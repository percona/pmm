import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getAgentVersions } from 'api/agents';
import { GetAgentVersionItem } from 'types/agent.types';

export const useAgentVersions = (
  options?: Partial<UseQueryOptions<GetAgentVersionItem[]>>
) =>
  useQuery({
    queryKey: ['agent/versions'],
    queryFn: getAgentVersions,
    ...options,
  });
