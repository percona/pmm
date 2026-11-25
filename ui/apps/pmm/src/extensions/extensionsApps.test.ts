import type { EnabledApp } from '@pmm-extensions/api';
import { MySqlIcon } from '@percona/peak-ui';
import {
  DEFAULT_EXTENSIONS_APP_ICON,
  extensionsAppPath,
  toExtensionsAppSlug,
  toExtensionsApps,
} from './extensionsApps';

const listed = (overrides: Partial<EnabledApp>): EnabledApp => ({
  app_key: 'pg_backups',
  blocking_dependencies: [],
  custom_ui: false,
  display_name: 'PostgreSQL Backups',
  enabled: true,
  group: null,
  nav_icon: null,
  nav_order: 10,
  react_route: '/apps/pg_backups',
  sidebar: true,
  uri_path: '/pg_backups',
  ...overrides,
});

const keys = (apps: EnabledApp[]) =>
  toExtensionsApps(apps).map((app) => app.key);

describe('toExtensionsAppSlug', () => {
  it.each([
    ['atw', 'atw'],
    ['mysql_backups', 'mysql-backups'],
    ['alert_troubleshooting', 'alert-troubleshooting'],
  ])('turns %s into %s', (key, slug) => {
    expect(toExtensionsAppSlug(key)).toBe(slug);
  });
});

describe('extensionsAppPath', () => {
  it('mounts an app under /extensions', () => {
    expect(extensionsAppPath('pg-backups')).toBe('/extensions/pg-backups');
  });
});

describe('toExtensionsApps', () => {
  it('maps an app to the shape PMM owns', () => {
    expect(toExtensionsApps([listed({ nav_icon: 'mysql' })])).toEqual([
      {
        key: 'pg_backups',
        slug: 'pg-backups',
        label: 'PostgreSQL Backups',
        icon: MySqlIcon,
        order: 10,
        enabled: true,
        customUi: false,
        blockedBy: [],
      },
    ]);
  });

  it('gives an app without an icon key the default icon', () => {
    expect(toExtensionsApps([listed({ nav_icon: null })])[0].icon).toBe(
      DEFAULT_EXTENSIONS_APP_ICON
    );
  });

  it('drops child apps and apps that are not in the sidebar', () => {
    expect(
      keys([
        listed({ app_key: 'mysql_backups' }),
        listed({ app_key: 'mysql_backups/restore' }),
        listed({ app_key: 'inventory', sidebar: false }),
      ])
    ).toEqual(['mysql_backups']);
  });

  it('keeps disabled apps, so the route can say they are disabled', () => {
    expect(toExtensionsApps([listed({ enabled: false })])[0].enabled).toBe(
      false
    );
  });

  it('orders by nav order, then by label, with unordered apps last', () => {
    expect(
      keys([
        listed({ app_key: 'unordered', nav_order: null }),
        listed({
          app_key: 'topology',
          display_name: 'Topology',
          nav_order: 13,
        }),
        listed({ app_key: 'report', display_name: 'Report', nav_order: 13 }),
        listed({ app_key: 'mysql_backups', nav_order: 8 }),
        listed({ app_key: 'atw', nav_order: 3 }),
      ])
    ).toEqual(['atw', 'mysql_backups', 'report', 'topology', 'unordered']);
  });

  it('names the apps that block a disabled one by their labels', () => {
    const [app] = toExtensionsApps([
      listed({
        enabled: false,
        blocking_dependencies: ['inventory', 'unknown_app'],
      }),
      listed({
        app_key: 'inventory',
        display_name: 'Inventory',
        sidebar: false,
      }),
    ]);

    expect(app.blockedBy).toEqual(['Inventory', 'unknown_app']);
  });
});
