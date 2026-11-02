import { useNavigation } from 'contexts/navigation';
import { useSettings } from 'contexts/settings';
import { useTour } from 'contexts/tour';
import { useUser } from 'contexts/user';
import { useVersion } from 'hooks/api/useVersion';
import { useCallback, useEffect, useMemo, useRef } from 'react';
import type { NavItem } from 'types/navigation.types';
import { mergeIndex } from './search.rank';
import type { RemoteState, SearchIndex, SearchItem } from './search.types';
import { useDashboardItems } from './sources/dashboards.source';
import { useInventoryItems } from './sources/inventory.source';
import {
  flattenNavTree,
  technologyDashboards,
} from './sources/navigation.source';
import {
  actionItems,
  extraPages,
  helpItems,
  settingsItems,
  type StaticSourceContext,
} from './sources/static.source';

export const buildStaticIndex = (
  navTree: NavItem[],
  context: StaticSourceContext
): SearchItem[] => {
  const nav = flattenNavTree(navTree, { isAdmin: context.isAdmin });

  return mergeIndex([
    nav.pages,
    nav.actions,
    extraPages(context),
    settingsItems(context),
    actionItems(context),
    helpItems(context),
    technologyDashboards(),
    nav.starredDashboards,
  ]);
};

export const useSearchIndex = (open: boolean): SearchIndex => {
  const { navTree } = useNavigation();
  const { user } = useUser();
  const { settings } = useSettings();
  const { startTour } = useTour();
  // startTour changes with the drawer state, so the index calls the latest one rather than rebuilding.
  const startTourRef = useRef(startTour);
  const runTour = useCallback<typeof startTour>(
    (tourName) => startTourRef.current(tourName),
    []
  );
  const isAdmin = !!user?.isPMMAdmin;
  const isEditor = !!user?.isEditor || isAdmin;
  const { data: version } = useVersion({ enabled: open && isAdmin });
  const dashboards = useDashboardItems(open);
  const inventory = useInventoryItems(open && isAdmin);

  useEffect(() => {
    startTourRef.current = startTour;
  }, [startTour]);

  const staticItems = useMemo(
    () =>
      buildStaticIndex(navTree, {
        isAdmin,
        isEditor,
        settings,
        version,
        startTour: runTour,
      }),
    [navTree, isAdmin, isEditor, settings, version, runTour]
  );

  const dashboardItems = dashboards.data;
  const inventoryItems = isAdmin ? inventory.items : undefined;

  const items = useMemo(
    () => mergeIndex([staticItems, dashboardItems ?? [], inventoryItems ?? []]),
    [staticItems, dashboardItems, inventoryItems]
  );

  const { isLoading, isError, refetch } = dashboards;
  const dashboardsState = useMemo<RemoteState>(
    () => ({
      status: isLoading
        ? 'loading'
        : isError
          ? 'error'
          : dashboardItems
            ? 'ready'
            : 'idle',
      refetch: () => {
        void refetch();
      },
    }),
    [isLoading, isError, dashboardItems, refetch]
  );

  return {
    items,
    staticItems,
    dashboards: dashboardsState,
    inventory: inventory.state,
  };
};
