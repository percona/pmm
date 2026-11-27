import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { hasKubernetesData } from 'api/kubernetes';

export const useHasKubernetesData = (
  options?: Partial<UseQueryOptions<boolean>>
) =>
  useQuery({
    queryKey: ['kubernetes:has-data'],
    queryFn: () => hasKubernetesData(),
    ...options,
  });
