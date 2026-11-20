import {
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import { findActiveNavItem } from 'utils/navigation.utils';
import { ServiceType } from 'types/services.types';
import { UpdateStatus } from 'types/updates.types';
import {
  addAlerting,
  addConfiguration,
  addDashboards,
} from './navigation.utils';
import { NAV_CONFIGURATION } from './navigation.constants';

const childIds = (item: ReturnType<typeof addAlerting>) =>
  (item.children || []).map((c) => c.id);

describe('addAlerting', () => {
  it('always includes alert rules', () => {
    expect(childIds(addAlerting())).toContain('alerts-rules');
  });

  it('does not add user-gated pages without a user', () => {
    const ids = childIds(addAlerting(true, true));

    expect(ids).toContain('alerts-rules');
    expect(ids).toContain('alerts-status');
    expect(ids).not.toContain('alerts-silences');
    expect(ids).not.toContain('alerts-groups');
    expect(ids).not.toContain('alerts-contact-points');
    expect(ids).not.toContain('alerts-policies');
    expect(ids).not.toContain('alerts-settings');
  });

  it('adds silences and groups for authenticated user when UA is off', () => {
    const ids = childIds(addAlerting(false, false, TEST_USER_VIEWER));

    expect(ids).toContain('alerts-silences');
    expect(ids).toContain('alerts-groups');
    expect(ids).not.toContain('alerts-contact-points');
    expect(ids).not.toContain('alerts-policies');
    expect(ids).not.toContain('alerts-settings');
  });

  it('adds contact points and policies for authenticated user when UA is on', () => {
    const ids = childIds(addAlerting(false, true, TEST_USER_VIEWER));

    expect(ids).toContain('alerts-silences');
    expect(ids).toContain('alerts-groups');
    expect(ids).toContain('alerts-contact-points');
    expect(ids).toContain('alerts-policies');
  });

  it('adds alert settings only for PMM admin when UA is on', () => {
    expect(childIds(addAlerting(false, true, TEST_USER_ADMIN))).toContain(
      'alerts-settings'
    );
    expect(childIds(addAlerting(false, true, TEST_USER_VIEWER))).not.toContain(
      'alerts-settings'
    );
    expect(childIds(addAlerting(false, false, TEST_USER_ADMIN))).not.toContain(
      'alerts-settings'
    );
  });

  it('adds status when alerting is enabled and templates only for editors', () => {
    expect(childIds(addAlerting(true, false))).toContain('alerts-status');
    expect(childIds(addAlerting(false, false))).not.toContain('alerts-status');

    expect(childIds(addAlerting(true, false, TEST_USER_EDITOR))).toContain(
      'alerts-templates'
    );
    expect(childIds(addAlerting(true, false, TEST_USER_VIEWER))).not.toContain(
      'alerts-templates'
    );
    expect(childIds(addAlerting(false, false, TEST_USER_EDITOR))).not.toContain(
      'alerts-templates'
    );
  });
});

describe('addConfiguration', () => {
  const updates = (item: ReturnType<typeof addConfiguration>) =>
    item.children?.find((child) => child.id === 'updates');

  it('describes the installed version without changing the shared item', () => {
    const item = addConfiguration(UpdateStatus.Pending, {
      installed: { version: '3.5.0' },
      updateAvailable: false,
    } as Parameters<typeof addConfiguration>[1]);

    expect(updates(item)).toMatchObject({
      secondaryText: 'v3.5.0, up to date',
      badge: { label: 'New' },
    });
    expect(updates(NAV_CONFIGURATION)?.secondaryText).toBeUndefined();
    expect(updates(NAV_CONFIGURATION)?.badge).toBeUndefined();
  });

  it('keeps a two-digit minor version and drops a build suffix', () => {
    const item = addConfiguration(UpdateStatus.Pending, {
      installed: { version: '3.10.0-v3-2d7d9c98f' },
      latest: { version: '3.11.0' },
      updateAvailable: true,
    } as Parameters<typeof addConfiguration>[1]);

    expect(updates(item)?.secondaryText).toBe('v3.10.0 → v3.11.0');
  });

  it('says when only the clients need an update', () => {
    const item = addConfiguration(UpdateStatus.UpdateClients, {
      installed: { version: '3.11.0' },
      updateAvailable: false,
    } as Parameters<typeof addConfiguration>[1]);

    expect(updates(item)?.secondaryText).toBe('Clients out of date');
  });

  it('drops the text of an earlier call', () => {
    addConfiguration(UpdateStatus.UpToDate, {
      installed: { version: '3.5.0' },
      updateAvailable: false,
    } as Parameters<typeof addConfiguration>[1]);

    expect(
      updates(addConfiguration(UpdateStatus.UpToDate))?.secondaryText
    ).toBeUndefined();
  });
});

describe('addDashboards', () => {
  it('lists the dashboards browser, then the Grafana pages for a user', () => {
    expect(childIds(addDashboards([], TEST_USER_VIEWER))).toEqual([
      'dashboards-browse',
      'dashboards-shared',
      'dashboards-playlists',
      'dashboards-snapshots',
      'dashboards-library-panels',
    ]);
  });

  it('lists only the dashboards browser without a user', () => {
    expect(childIds(addDashboards([]))).toEqual(['dashboards-browse']);
  });

  it('adds ProxySQL and HAProxy only while PMM monitors them', () => {
    expect(childIds(addDashboards([ServiceType.mysql]))).not.toContain(
      'proxysql'
    );
    expect(
      childIds(addDashboards([ServiceType.proxysql, ServiceType.haproxy]))
    ).toEqual(['dashboards-browse', 'proxysql', 'haproxy']);
  });

  it.each([
    ['/graph/dashboards', 'dashboards-browse'],
    ['/graph/dashboards/f/abc/experimental', 'dashboards-browse'],
    ['/graph/dashboard/public', 'dashboards-shared'],
    ['/graph/dashboard/snapshots', 'dashboards-snapshots'],
    ['/graph/playlists', 'dashboards-playlists'],
    ['/graph/playlists/play/1', 'dashboards-playlists'],
    ['/graph/library-panels', 'dashboards-library-panels'],
    [
      '/graph/d/proxysql-instance-summary/proxysql-instance-summary',
      'proxysql',
    ],
    ['/graph/dashboard/new', 'dashboards'],
    ['/graph/dashboard/import', 'dashboards'],
  ])('highlights %s as %s', (pathname, id) => {
    const dashboards = addDashboards([ServiceType.proxysql], TEST_USER_ADMIN);

    expect(findActiveNavItem(dashboards, pathname)?.id).toBe(id);
  });
});
