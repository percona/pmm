import { useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ENABLED_APPS_QUERY_KEY } from '@pmm-extensions/api';
import { getExtensionsApps } from 'api/extensionsApps';
import { useSettings } from 'contexts/settings';
import { useUser } from 'contexts/user';
import { toExtensionsApps } from 'extensions/extensionsApps';

// Under the side-car's own `['apps']` key, so enabling or disabling an app
// through `useAdminApps` refreshes this list too.
export const EXTENSIONS_APPS_QUERY_KEY = [
  ...ENABLED_APPS_QUERY_KEY,
  'pmm',
] as const;

/**
 * The PMM Extensions apps, for the Apps section and the generic app route.
 *
 * Fetched once per page load while Extensions is enabled, and kept for the
 * session: a refetch happens only when an Extensions page mounts
 * ({@link useRefreshExtensionsApps}) or the query is invalidated. A failure is
 * not retried, since the navigation has a fallback for it.
 */
export const useExtensionsApps = () => {
  const { user } = useUser();
  const { settings } = useSettings();

  return useQuery({
    queryKey: EXTENSIONS_APPS_QUERY_KEY,
    queryFn: async () => toExtensionsApps(await getExtensionsApps()),
    enabled: !!user && !!settings?.extensionsEnabled,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    retry: false,
  });
};

/**
 * Fetch the app list again when the calling component mounts, and whenever
 * `appSlug` changes: the router keeps one Extensions page mounted while the
 * user moves from one app to the next.
 *
 * A fetch still in flight is cancelled first. Invalidating would otherwise join
 * it, and a first fetch that fails would then leave no list behind it.
 */
export const useRefreshExtensionsApps = (appSlug?: string) => {
  const queryClient = useQueryClient();

  useEffect(() => {
    const queryKey = EXTENSIONS_APPS_QUERY_KEY;

    void queryClient
      .cancelQueries({ queryKey })
      .then(() => queryClient.invalidateQueries({ queryKey }));
  }, [queryClient, appSlug]);
};
