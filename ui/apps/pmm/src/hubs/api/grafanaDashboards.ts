import { grafanaApi } from 'api/api';
import { safeLocalStorage } from 'utils/storage.utils';

export interface GrafanaDashboardHit {
  uid: string;
  title: string;
  /** Carries Grafana's sub path: /graph/d/<uid>/<slug>. */
  url: string;
  type?: string;
  folderUid?: string;
  folderTitle?: string;
  tags?: string[];
}

export interface DashboardSearchResult {
  uid: string;
  title: string;
  url: string;
  folderUid?: string;
}

/** `GET /graph/api/search?type=dash-db&query=…`: dashboards whose title matches. */
export const searchDashboards = async (
  query: string
): Promise<DashboardSearchResult[]> => {
  const res = await grafanaApi.get<GrafanaDashboardHit[]>('/search', {
    params: { type: 'dash-db', query },
  });

  return mapDashboardHits(res.data);
};

export const mapDashboardHits = (
  hits: GrafanaDashboardHit[]
): DashboardSearchResult[] =>
  hits
    .filter((hit) => !hit.type || hit.type === 'dash-db')
    .map(({ uid, title, url, folderUid }) => ({
      uid,
      title,
      url,
      ...(folderUid ? { folderUid } : {}),
    }));

/** Grafana's own "recently viewed" list, kept per organisation in this origin's storage. */
export const impressionsKey = (orgId: number) =>
  `dashboard_impressions-${orgId}`;

/** Dashboard uids the person opened, newest first; empty when missing, malformed or unreadable. */
export const readImpressions = (orgId: number): string[] => {
  try {
    const stored: unknown = JSON.parse(
      safeLocalStorage.getItem(impressionsKey(orgId)) ?? '[]'
    );

    return Array.isArray(stored)
      ? stored.filter((uid): uid is string => typeof uid === 'string' && !!uid)
      : [];
  } catch {
    return [];
  }
};
