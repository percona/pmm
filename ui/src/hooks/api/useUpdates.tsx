import { checkForUpdates } from 'api/updates';
import { useQuery } from '@tanstack/react-query';

export const useCheckUpdates = () =>
  useQuery({
    queryKey: ['checkUpdates'],
    queryFn: async () => {
      try {
        return await checkForUpdates();
      } catch (error) {
        return await checkForUpdates({
          force: false,
          onlyInstalledVersion: true,
        });
      }
    },
  });
