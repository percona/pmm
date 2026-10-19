import { grafanaApi } from 'api/api';
import {
  impressionsKey,
  mapDashboardHits,
  readImpressions,
  searchDashboards,
} from './grafanaDashboards';

// Two hits as Grafana's `GET /graph/api/search?type=dash-db&query=MongoDB` returns them.
const HITS = [
  {
    id: 41,
    uid: 'mongodb-replicaset-summary',
    title: 'MongoDB ReplSet Summary',
    uri: 'db/mongodb-replset-summary',
    url: '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary',
    type: 'dash-db',
    tags: ['MongoDB', 'Percona'],
    isStarred: false,
    folderUid: 'eg1pcc3mr38cgd',
    folderTitle: 'MongoDB',
  },
  {
    id: 12,
    uid: 'prometheus-status',
    title: 'Prometheus Exporter Status',
    url: '/graph/d/prometheus-status/prometheus-exporter-status',
    type: 'dash-db',
    folderUid: 'dg1pcc3thbdvka',
    folderTitle: 'Insight',
  },
];

describe('mapDashboardHits', () => {
  it('keeps uid, title, url and folder', () => {
    expect(mapDashboardHits(HITS)).toEqual([
      {
        uid: 'mongodb-replicaset-summary',
        title: 'MongoDB ReplSet Summary',
        url: '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary',
        folderUid: 'eg1pcc3mr38cgd',
      },
      {
        uid: 'prometheus-status',
        title: 'Prometheus Exporter Status',
        url: '/graph/d/prometheus-status/prometheus-exporter-status',
        folderUid: 'dg1pcc3thbdvka',
      },
    ]);
  });

  it('drops folders and leaves out a missing folder', () => {
    expect(
      mapDashboardHits([
        {
          uid: 'f',
          title: 'MongoDB',
          url: '/graph/dashboards/f/f',
          type: 'dash-folder',
        },
        { uid: 'g', title: 'General one', url: '/graph/d/g' },
      ])
    ).toEqual([{ uid: 'g', title: 'General one', url: '/graph/d/g' }]);
  });
});

describe('searchDashboards', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('asks Grafana for dashboards matching the query', async () => {
    const get = vi
      .spyOn(grafanaApi, 'get')
      .mockResolvedValue({ data: HITS } as never);

    await expect(searchDashboards('MongoDB')).resolves.toHaveLength(2);
    expect(get).toHaveBeenCalledWith('/search', {
      params: { type: 'dash-db', query: 'MongoDB' },
    });
  });
});

describe('readImpressions', () => {
  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('reads Grafana’s recently viewed uids for the organisation', () => {
    localStorage.setItem(
      impressionsKey(1),
      JSON.stringify(['mongodb-oplog-details', 'mongodb-instance-summary'])
    );

    expect(readImpressions(1)).toEqual([
      'mongodb-oplog-details',
      'mongodb-instance-summary',
    ]);
    expect(readImpressions(2)).toEqual([]);
  });

  it('uses the key Grafana writes', () => {
    expect(impressionsKey(3)).toBe('dashboard_impressions-3');
  });

  it('is empty when the key is missing', () => {
    expect(readImpressions(1)).toEqual([]);
  });

  it('is empty for invalid JSON or something other than a list', () => {
    localStorage.setItem(impressionsKey(1), '{not json');
    expect(readImpressions(1)).toEqual([]);

    localStorage.setItem(impressionsKey(1), JSON.stringify({ uid: 'x' }));
    expect(readImpressions(1)).toEqual([]);
  });

  it('skips entries that are not uids, such as old numeric ids', () => {
    localStorage.setItem(
      impressionsKey(1),
      JSON.stringify([7, '', 'abc', null])
    );

    expect(readImpressions(1)).toEqual(['abc']);
  });

  it('is empty when storage throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('SecurityError');
    });

    expect(readImpressions(1)).toEqual([]);
  });
});
