import { useQuery } from '@tanstack/react-query';
import { getReadonlySettings } from '../api/settings';
import { QUERY_KEYS } from '../query/query.constants';

export const useReadonlySettings = () =>
  useQuery({
    queryKey: QUERY_KEYS.readonlySettings,
    queryFn: getReadonlySettings,
  });
