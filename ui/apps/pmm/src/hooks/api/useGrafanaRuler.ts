import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getRulerRule } from 'api/ruler';
import { GrafanaRulerRuleDTO } from 'types/ruler.types';

export const useGrafanaRulerRule = (
  uid: string,
  options?: Omit<UseQueryOptions<GrafanaRulerRuleDTO>, 'queryKey' | 'queryFn'>
) =>
  useQuery({
    queryKey: ['grafana-ruler-rule', uid],
    queryFn: async () => getRulerRule(uid),
    // Rule definitions change rarely; avoids a refetch per prev/next step
    // through alerts of the same rule.
    staleTime: 30_000,
    ...options,
  });
