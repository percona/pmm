import { PMM_NEW_NAV_PATH } from 'lib/constants';
import type { RouteObject } from 'react-router-dom';
import { HUB_PATHS, hubClustersPath } from './hubs.paths';

const relative = (path: string) => path.slice(PMM_NEW_NAV_PATH.length + 1);

// Two routes per hub, relative to the PMM_NEW_NAV_PATH parent: the hub and its Clusters page. Hub code loads in its own chunk on first visit.
export const hubRoutes: RouteObject[] = Object.values(HUB_PATHS).flatMap(
  (path) => [
    {
      path: relative(path),
      lazy: async () => {
        const { HubRoute } = await import('./template/HubRoute');

        return { Component: HubRoute };
      },
      // Renders the sidebar on a first visit straight to a hub while its chunk loads.
      HydrateFallback: () => null,
    },
    {
      path: relative(hubClustersPath(path)),
      lazy: async () => {
        const { ClustersPlaceholderPage } =
          await import('./template/ClustersPlaceholderPage');

        return { Component: ClustersPlaceholderPage };
      },
      HydrateFallback: () => null,
    },
  ]
);
