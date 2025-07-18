import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { listAdvisors } from 'api/advisors';
import { Advisor } from 'types/advisors.types';

export const useAdvisors = (options?: UseQueryOptions<Advisor[]>) =>
  useQuery({
    queryKey: ['advisors'],
    queryFn: () => listAdvisors(),
    ...options,
  });
