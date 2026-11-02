import { TEST_USER_ADMIN, TEST_USER_EDITOR } from 'utils/testStubs';
import {
  getRecentKey,
  pushRecent,
  readRecent,
  resolveRecent,
} from './search.recent';
import { testItem } from './search.testFixtures';

const KEY = getRecentKey(TEST_USER_ADMIN);

const page = (n: number) => testItem(`Page ${n}`, { to: `/page/${n}` });

describe('search.recent', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('keeps one list per org and user', () => {
    expect(KEY).toBe('pmm-ui.search.recent.1.1');
    expect(getRecentKey({ ...TEST_USER_ADMIN, orgId: 2 })).toBe(
      'pmm-ui.search.recent.2.1'
    );
    expect(getRecentKey(TEST_USER_EDITOR)).toBe('pmm-ui.search.recent.1.2');
  });

  it('moves a revisited entry to the front', () => {
    pushRecent(KEY, page(1), 1);
    pushRecent(KEY, page(2), 2);
    pushRecent(KEY, page(1), 3);

    expect(readRecent(KEY).map((entry) => [entry.label, entry.at])).toEqual([
      ['Page 1', 3],
      ['Page 2', 2],
    ]);
    expect(JSON.parse(localStorage.getItem(KEY) ?? '').v).toBe(1);
  });

  it('stores at most 20 and shows at most 5', () => {
    for (let n = 0; n < 25; n++) {
      pushRecent(KEY, page(n));
    }

    const entries = readRecent(KEY);

    expect(entries).toHaveLength(20);
    expect(entries[0].label).toBe('Page 24');
    expect(resolveRecent(entries, [], '/elsewhere')).toHaveLength(5);
  });

  it('records only destinations', () => {
    pushRecent(KEY, testItem('Act', { kind: 'action', to: '/graph/x' }));
    pushRecent(KEY, testItem('All', { kind: 'link', to: '/graph/y' }));
    pushRecent(KEY, testItem('Tour', { kind: 'action' }));
    pushRecent(KEY, testItem('Docs', { kind: 'help', href: 'https://x' }));

    expect(readRecent(KEY)).toEqual([]);
  });

  it('prefers live items and drops the current page', () => {
    pushRecent(KEY, page(1));
    pushRecent(KEY, page(2));
    const live = testItem('Page 1 renamed', {
      id: 'page:Page 1',
      to: '/page/1',
    });

    expect(
      resolveRecent(readRecent(KEY), [live], '/page/2').map(
        (item) => item.label
      )
    ).toEqual(['Page 1 renamed']);
    expect(resolveRecent(readRecent(KEY), [], '/page/1')[0]).toMatchObject({
      id: 'page:Page 2',
      label: 'Page 2',
      to: '/page/2',
      group: 'recent',
    });
  });

  it('drops the current page by its destination and shows each destination once', () => {
    const overview = testItem('Overview', {
      id: 'page:home-page',
      to: '/graph/',
      dedupeKey: 'dashboard:pmm-home',
    });
    const homeDashboard = testItem('Home Dashboard', {
      id: 'dashboard:pmm-home',
      kind: 'dashboard',
      to: '/graph/d/pmm-home/home-dashboard',
    });
    pushRecent(KEY, homeDashboard);
    pushRecent(KEY, overview);
    pushRecent(KEY, page(1));
    const shown = (pathname: string) =>
      resolveRecent(readRecent(KEY), [overview], pathname).map(
        (item) => item.label
      );

    expect(shown('/graph/d/pmm-home/home-dashboard')).toEqual(['Page 1']);
    expect(shown('/graph')).toEqual(['Page 1']);
    expect(shown('/page/1')).toEqual(['Overview']);
  });

  it.each(['not json', '{"v":2,"items":[]}', '{"v":1,"items":{}}', '[1]'])(
    'reads %s as empty',
    (stored) => {
      localStorage.setItem(KEY, stored);

      expect(readRecent(KEY)).toEqual([]);
    }
  );

  it('skips malformed entries', () => {
    localStorage.setItem(
      KEY,
      JSON.stringify({
        v: 1,
        items: [
          { id: 'a', kind: 'page', label: 'A', to: '/a', at: 1 },
          { id: 'b', kind: 'action', label: 'B', to: '/b', at: 1 },
          { id: 'c', kind: 'page', label: 'C' },
          null,
        ],
      })
    );

    expect(readRecent(KEY).map((entry) => entry.id)).toEqual(['a']);
  });
});
