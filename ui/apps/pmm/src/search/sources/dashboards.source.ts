import { type QueryClient, useQuery } from '@tanstack/react-query';
import { grafanaApi } from 'api/api';
import { DASHBOARDS_QUERY_KEY, REMOTE_STALE_TIME } from '../search.constants';
import { searchItem } from '../search.keywords';
import { Messages } from '../search.messages';
import { carriesVariables } from '../search.text';
import type { SearchItem } from '../search.types';

export interface GrafanaSearchHit {
  uid: string;
  title: string;
  // Already carries Grafana's sub path, such as /graph/d/<uid>/<slug>.
  url: string;
  type: 'dash-db' | 'dash-folder';
  tags?: string[];
  isStarred?: boolean;
  folderTitle?: string;
  isDeleted?: boolean;
}

export const mapDashboardHits = (hits: GrafanaSearchHit[]): SearchItem[] =>
  hits.flatMap((hit) => {
    if (hit.isDeleted) {
      return [];
    }

    if (hit.type === 'dash-db') {
      return [
        searchItem({
          id: `dashboard:${hit.uid}`,
          kind: 'dashboard',
          group: 'dashboards',
          label: hit.title,
          description: [
            Messages.crumbs.dashboards,
            hit.folderTitle ?? Messages.crumbs.general,
          ],
          keywords: hit.tags ?? [],
          to: hit.url,
          carryVariables: carriesVariables(hit.url),
          starred: hit.isStarred === true,
          remote: true,
        }),
      ];
    }

    if (hit.type === 'dash-folder') {
      return [
        searchItem({
          id: `folder:${hit.uid}`,
          kind: 'folder',
          group: 'dashboards',
          label: hit.title,
          description: Messages.crumbs.folder,
          to: hit.url,
          remote: true,
        }),
      ];
    }

    return [];
  });

export const fetchDashboardItems = async () =>
  mapDashboardHits(
    (
      await grafanaApi.get<GrafanaSearchHit[]>('/search', {
        params: { limit: 5000 },
      })
    ).data
  );

export const useDashboardItems = (enabled: boolean) =>
  useQuery({
    queryKey: DASHBOARDS_QUERY_KEY,
    queryFn: fetchDashboardItems,
    staleTime: REMOTE_STALE_TIME,
    retry: 1,
    enabled,
  });

export const prefetchDashboardItems = (queryClient: QueryClient) =>
  queryClient.prefetchQuery({
    queryKey: DASHBOARDS_QUERY_KEY,
    queryFn: fetchDashboardItems,
    staleTime: REMOTE_STALE_TIME,
  });
