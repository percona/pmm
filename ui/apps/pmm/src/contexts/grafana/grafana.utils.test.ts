import { isMigratedPage } from './grafana.utils';

describe('isMigratedPage', () => {
  it.each([
    '/graph/settings/advanced-settings',
    '/graph/add-instance',
    '/graph/add-instance/rds',
  ])('treats %s as a native page', (src) =>
    expect(isMigratedPage(src)).toBe(true)
  );

  it.each(['/graph/inventory/services', '/graph/edit-instance/service-1'])(
    'leaves %s to Grafana',
    (src) => expect(isMigratedPage(src)).toBe(false)
  );
});
