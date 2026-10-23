import { NAV_MONGO, NAV_MYSQL } from 'contexts/navigation/navigation.constants';
import { MONGODB_HUB_PATH } from 'lib/constants';
import type { NavItem } from 'types/navigation.types';
import { HUB_PATHS, withHubPage } from './hubs.paths';

const OVERVIEW =
  '/graph/d/mongodb-instance-overview/mongodb-instances-overview';
const SUMMARY = '/graph/d/mongodb-instance-summary/mongodb-instance-summary';

describe('HUB_PATHS', () => {
  it('keys the MongoDB hub by the MongoDB sidebar entry', () => {
    expect(HUB_PATHS[NAV_MONGO.id]).toBe(MONGODB_HUB_PATH);
    expect(MONGODB_HUB_PATH).toBe('/mongodb');
  });
});

describe('withHubPage', () => {
  const mongo: NavItem = {
    id: NAV_MONGO.id,
    text: 'MongoDB',
    url: OVERVIEW,
    matches: [OVERVIEW, SUMMARY],
  };

  it('opens the hub and stays highlighted on the dashboards', () => {
    expect(withHubPage(mongo)).toEqual({
      ...mongo,
      url: MONGODB_HUB_PATH,
      matches: [MONGODB_HUB_PATH, OVERVIEW, SUMMARY],
    });
  });

  it('works for an entry without matches', () => {
    expect(withHubPage({ id: NAV_MONGO.id, url: OVERVIEW }).matches).toEqual([
      MONGODB_HUB_PATH,
    ]);
  });

  it('returns an entry without a hub unchanged', () => {
    const mysql: NavItem = { id: NAV_MYSQL.id, url: NAV_MYSQL.url };

    expect(withHubPage(mysql)).toBe(mysql);
  });

  it('leaves the entry it is given untouched', () => {
    const before = structuredClone(mongo);
    withHubPage(mongo);

    expect(mongo).toEqual(before);
  });
});
