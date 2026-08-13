import type { UseQueryOptions } from '@tanstack/react-query';
import { useQuery } from '@tanstack/react-query';
import { listAdvisors } from 'api/advisors';
import type { Advisor } from 'types/advisors.types';

export const useAdvisors = (options?: Partial<UseQueryOptions<Advisor[]>>) =>
  useQuery({
    queryKey: ['advisors'],
    queryFn: () => listAdvisors(),
    ...options,
  });
