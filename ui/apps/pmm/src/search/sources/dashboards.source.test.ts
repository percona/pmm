import { QueryClient } from '@tanstack/react-query';
import { grafanaApi } from 'api/api';
import {
  fetchDashboardItems,
  mapDashboardHits,
  prefetchDashboardItems,
  type GrafanaSearchHit,
} from './dashboards.source';

const HITS: GrafanaSearchHit[] = [
  {
    uid: 'node-disk',
    title: 'Disk details',
    url: '/graph/d/node-disk/disk-details',
    type: 'dash-db',
    tags: ['os', 'disk'],
    isStarred: true,
    folderTitle: 'OS',
  },
  {
    uid: 'custom',
    title: 'Custom',
    url: '/graph/d/custom/custom',
    type: 'dash-db',
  },
  {
    uid: 'os-folder',
    title: 'OS',
    url: '/graph/dashboards/f/os-folder/os',
    type: 'dash-folder',
  },
  {
    uid: 'gone',
    title: 'Gone',
    url: '/graph/d/gone/gone',
    type: 'dash-db',
    isDeleted: true,
  },
];

describe('dashboards.source', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('maps dashboards and folders as remote items and drops deleted ones', () => {
    const [disk, custom, folder, ...rest] = mapDashboardHits(HITS);

    expect(rest).toEqual([]);
    expect(disk).toMatchObject({
      id: 'dashboard:node-disk',
      kind: 'dashboard',
      group: 'dashboards',
      label: 'Disk details',
      description: ['Dashboards', 'OS'],
      keywords: ['os', 'disk'],
      to: '/graph/d/node-disk/disk-details',
      carryVariables: true,
      starred: true,
      remote: true,
      dedupeKey: 'dashboard:node-disk',
    });
    expect(custom).toMatchObject({
      description: ['Dashboards', 'General'],
      keywords: [],
      starred: false,
    });
    expect(folder).toMatchObject({
      id: 'folder:os-folder',
      kind: 'folder',
      group: 'dashboards',
      description: 'Folder',
      to: '/graph/dashboards/f/os-folder/os',
      remote: true,
    });
  });

  it('fetches every dashboard once and returns mapped items', async () => {
    const get = vi
      .spyOn(grafanaApi, 'get')
      .mockResolvedValue({ data: HITS.slice(0, 1) });

    await expect(fetchDashboardItems()).resolves.toEqual([
      expect.objectContaining({ id: 'dashboard:node-disk' }),
    ]);
    expect(get).toHaveBeenCalledWith('/search', { params: { limit: 5000 } });
  });

  it('prefetches into the search cache only while stale', async () => {
    const get = vi.spyOn(grafanaApi, 'get').mockResolvedValue({ data: HITS });
    const queryClient = new QueryClient();

    await prefetchDashboardItems(queryClient);
    await prefetchDashboardItems(queryClient);

    expect(get).toHaveBeenCalledTimes(1);
    expect(queryClient.getQueryData(['search', 'dashboards'])).toHaveLength(3);
  });
});
