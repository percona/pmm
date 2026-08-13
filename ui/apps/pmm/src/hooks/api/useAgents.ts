import type { UseQueryOptions } from '@tanstack/react-query';
import { useQuery } from '@tanstack/react-query';
import { getAgentVersions } from 'api/agents';
import type { GetAgentVersionItem } from 'types/agent.types';

export const useAgentVersions = (
  options?: Partial<UseQueryOptions<GetAgentVersionItem[]>>
) =>
  useQuery({
    queryKey: ['agent/versions'],
    queryFn: getAgentVersions,
    ...options,
  });
