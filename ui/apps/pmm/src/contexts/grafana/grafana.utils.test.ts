import { isMigratedPage } from './grafana.utils';

describe('isMigratedPage', () => {
  it.each([
    '/graph/settings/advanced-settings',
    '/graph/inventory/services',
    '/graph/edit-instance/service-1',
    '/graph/add-instance',
    '/graph/add-instance/rds',
    '/graph/advisors',
    '/graph/advisors/insights',
    '/graph/pmm-database-checks/failed-checks',
  ])('treats %s as migrated', (src) => {
    expect(isMigratedPage(src)).toBe(true);
  });

  it('leaves Grafana pages in the iframe', () => {
    expect(isMigratedPage('/graph/d/pmm-home/home-dashboard')).toBe(false);
  });
});
