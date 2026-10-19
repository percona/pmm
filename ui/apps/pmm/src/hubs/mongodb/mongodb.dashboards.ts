import type { HubDashboardLink, HubDashboards } from 'hubs/template/hub.types';
import {
  listDashboard,
  MONGO_DASHBOARDS,
  type MongoDashboardId,
} from './mongodb.doors';
import { MONGODB_HUB_MESSAGES, Messages } from './mongodb.messages';
import type { ClusterModel } from './mongodb.topology';
import type { DashboardHit, MongoSnapshot } from './mongodb.types';

const RECENT_MAX = 5;

const ALL_QUERY_PATH = '/graph/dashboards?query=MongoDB';

const uidOf = (id: MongoDashboardId) => MONGO_DASHBOARDS[id].split('/')[0];

const LABEL_BY_UID = new Map(
  (Object.keys(MONGO_DASHBOARDS) as MongoDashboardId[]).map((id) => [
    uidOf(id),
    Messages.dashboards[id],
  ])
);

export interface DashboardsInput {
  clusters: ClusterModel[];
  /** Null when the catalog failed. */
  dashboards: MongoSnapshot['dashboards'];
  /** A backup schedule, an artifact or PBM configured anywhere shown. */
  hasBackups: boolean;
  filter: string | null;
}

/** The folder most catalog hits share, which is where PMM provisions its MongoDB dashboards. */
export const mainFolder = (catalog: DashboardHit[]): string | undefined => {
  const counts = new Map<string, number>();

  catalog.forEach(({ folderUid }) => {
    if (folderUid) {
      counts.set(folderUid, (counts.get(folderUid) ?? 0) + 1);
    }
  });

  return [...counts.entries()].sort(
    (a, b) => b[1] - a[1] || a[0].localeCompare(b[0])
  )[0]?.[0];
};

const link = (
  id: MongoDashboardId,
  filter: string | null
): HubDashboardLink => ({
  uid: uidOf(id),
  label: Messages.dashboards[id],
  to: listDashboard(id, filter),
});

/** What to recommend for what PMM sees running. */
export const recommendedIds = ({
  clusters,
  hasBackups,
}: Pick<DashboardsInput, 'clusters' | 'hasBackups'>): MongoDashboardId[] => {
  if (!clusters.length) {
    return [];
  }

  return [
    'instancesOverview',
    'instanceSummary',
    ...(clusters.some((cluster) => cluster.kind !== 'standalone')
      ? (['replsetSummary', 'oplogDetails'] as const)
      : []),
    ...(clusters.some((cluster) => cluster.kind === 'sharded')
      ? (['clusterSummary', 'routerSummary'] as const)
      : []),
    ...(hasBackups ? (['backupStatus'] as const) : []),
  ];
};

export const deriveDashboards = ({
  clusters,
  dashboards,
  hasBackups,
  filter,
}: DashboardsInput): HubDashboards => {
  const hits = dashboards?.catalog ?? [];
  const folderUid = mainFolder(hits);
  const catalog = folderUid
    ? hits.filter((hit) => hit.folderUid === folderUid)
    : hits;
  const titles = new Map(catalog.map((hit) => [hit.uid, hit.title]));
  const seen = new Set<string>();
  const recent = (dashboards?.recent ?? [])
    .filter(({ uid }) => {
      const fresh = titles.has(uid) && !seen.has(uid);

      seen.add(uid);

      return fresh;
    })
    .slice(0, RECENT_MAX)
    .map(
      ({ uid, visitedAt }): HubDashboardLink => ({
        uid,
        label: LABEL_BY_UID.get(uid) ?? titles.get(uid) ?? uid,
        to: `/graph/d/${uid}`,
        ...(visitedAt ? { visitedAt } : {}),
      })
    );

  return {
    recommended: recommendedIds({ clusters, hasBackups }).map((id) =>
      link(id, filter)
    ),
    forDevelopers: clusters.length
      ? [link('collectionsOverview', filter), link('unusedIndexes', filter)]
      : [],
    recent,
    all: {
      label: MONGODB_HUB_MESSAGES.dashboards.all,
      to: folderUid ? `/graph/dashboards/f/${folderUid}/` : ALL_QUERY_PATH,
      ...(dashboards ? { count: catalog.length } : {}),
    },
  };
};
