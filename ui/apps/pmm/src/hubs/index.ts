import { PMM_NEW_NAV_PATH } from 'lib/constants';
import type { RouteObject } from 'react-router-dom';
import { HUB_PATHS } from './hubs.paths';

// One route per hub, relative to the PMM_NEW_NAV_PATH parent; hub code loads in its own chunk on first visit.
export const hubRoutes: RouteObject[] = Object.values(HUB_PATHS).map(
  (path) => ({
    path: path.slice(PMM_NEW_NAV_PATH.length + 1),
    lazy: async () => {
      const { HubRoute } = await import('./template/HubRoute');

      return { Component: HubRoute };
    },
    // Renders the sidebar on a first visit straight to a hub while its chunk loads.
    HydrateFallback: () => null,
  })
);
