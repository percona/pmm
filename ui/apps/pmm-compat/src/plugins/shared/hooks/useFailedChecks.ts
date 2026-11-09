import { useQuery } from '@tanstack/react-query';
import { getFailedServices } from '../api/advisors';
import { QUERY_KEYS } from '../query/query.constants';

export const useFailedChecks = (enabled: boolean) =>
  useQuery({
    queryKey: QUERY_KEYS.failedServices,
    queryFn: getFailedServices,
    enabled,
  });
