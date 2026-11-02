import { useQueryClient } from '@tanstack/react-query';
import { useUser } from 'contexts/user';
import { useEffect, useRef } from 'react';
import { useLocation } from 'react-router-dom';
import { DASHBOARDS_QUERY_KEY, REMOTE_STALE_TIME } from './search.constants';
import { getRecentKey, isRecordable, pushRecent } from './search.recent';
import { canonicalKey } from './search.text';
import type { SearchItem } from './search.types';
import { fetchDashboardItems } from './sources/dashboards.source';

const findRecordable = (items: SearchItem[], key: string) =>
  items.find((item) => item.dedupeKey === key && isRecordable(item));

// Records every page visited, so Recent is useful before search is first opened.
export const useRecentVisits = (staticItems: SearchItem[]) => {
  const { pathname } = useLocation();
  const { user } = useUser();
  const queryClient = useQueryClient();
  const recentKey = user ? getRecentKey(user) : undefined;
  // Dashboards missing from a cached list, each refetched for once.
  const refetched = useRef(new Set<string>());

  useEffect(() => {
    const target = canonicalKey(pathname);

    if (!recentKey || !target) {
      return;
    }

    const known = findRecordable(staticItems, target);

    if (known) {
      pushRecent(recentKey, known);
      return;
    }

    if (!target.startsWith('dashboard:')) {
      return;
    }

    let cancelled = false;

    const resolveDashboard = async () => {
      const wasCached =
        queryClient.getQueryData(DASHBOARDS_QUERY_KEY) !== undefined;
      const dashboard = findRecordable(
        await queryClient.ensureQueryData({
          queryKey: DASHBOARDS_QUERY_KEY,
          queryFn: fetchDashboardItems,
          staleTime: REMOTE_STALE_TIME,
        }),
        target
      );

      if (dashboard || !wasCached || refetched.current.has(target)) {
        return dashboard;
      }

      // The cached list can predate the dashboard, such as one just created or imported.
      refetched.current.add(target);

      return findRecordable(
        await queryClient.fetchQuery({
          queryKey: DASHBOARDS_QUERY_KEY,
          queryFn: fetchDashboardItems,
          staleTime: 0,
        }),
        target
      );
    };

    resolveDashboard()
      .then((dashboard) => {
        if (!cancelled && dashboard) {
          pushRecent(recentKey, dashboard);
        }
      })
      .catch(() => {
        // Unresolvable visits are skipped.
      });

    return () => {
      cancelled = true;
    };
  }, [pathname, recentKey, staticItems, queryClient]);
};
