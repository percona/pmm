import { useQuery } from '@tanstack/react-query';
import { getAgentVersions } from 'api/agents';

export const useAgentVersions = () =>
  useQuery({
    queryKey: ['agent/versions'],
    queryFn: getAgentVersions,
  });
