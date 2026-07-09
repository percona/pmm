import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getGrafanaRulerRule } from 'api/grafana-ruler';
import { GrafanaRulerRuleDTO } from 'types/grafana-ruler.types';

export const useGrafanaRulerRule = (
  uid: string,
  options?: Omit<UseQueryOptions<GrafanaRulerRuleDTO>, 'queryKey' | 'queryFn'>
) =>
  useQuery({
    queryKey: ['grafana-ruler-rule', uid],
    queryFn: async () => getGrafanaRulerRule(uid),
    ...options,
  });
