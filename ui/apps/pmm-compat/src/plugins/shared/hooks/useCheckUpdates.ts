import { useQuery } from '@tanstack/react-query';
import { checkForUpdates } from '../api/updates';
import { QUERY_KEYS } from '../query/query.constants';

export const useCheckUpdates = (enabled: boolean) =>
  useQuery({
    queryKey: QUERY_KEYS.checkUpdates,
    queryFn: async () => {
      try {
        return await checkForUpdates({ force: false });
      } catch {
        return checkForUpdates({ force: false, only_installed_version: true });
      }
    },
    enabled,
  });
