import { isMigratedPage } from './grafana.utils';

describe('isMigratedPage', () => {
  it.each([
    '/graph/settings/advanced-settings',
    '/graph/inventory/services',
    '/graph/edit-instance/service-1',
    '/graph/add-instance',
    '/graph/add-instance/rds',
  ])('treats %s as a native page', (src) =>
    expect(isMigratedPage(src)).toBe(true)
  );

  it('leaves other Grafana pages to Grafana', () =>
    expect(isMigratedPage('/graph/d/pmm-home')).toBe(false));
});
