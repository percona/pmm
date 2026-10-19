import { describe } from 'vitest';
import { screen, within } from '@testing-library/react';
import { closestTo, hasPeakHub, renderHub } from '../../testUtils';
import { minutesAgo, READY, testDoor, testView } from '../hub.testFixtures';
import type { HubBackupRow, HubBackups, HubMessages } from '../hub.types';
import { BackupsSection } from './BackupsSection';

const MESSAGES = {
  backups: {
    emptyTitle: 'No MongoDB backups yet',
    emptyDescription:
      'Set up a scheduled backup to keep a copy of each cluster.',
    noServices: 'Backups start once a MongoDB service is monitored.',
  },
} as HubMessages;

const row = (
  cluster: string,
  overrides: Partial<HubBackupRow> = {}
): HubBackupRow => ({
  id: cluster,
  cluster,
  lastGood: { at: minutesAgo(360), level: 'ok' },
  pitr: 'on',
  nextRun: '2026-10-20T03:00:00',
  doors: {
    cluster: testDoor(
      cluster,
      `/graph/d/mongodb-backup-details?var-cluster=${cluster}`
    ),
    lastGood: testDoor('Last good backup', '/graph/backup/inventory', {
      context: `of ${cluster}`,
    }),
    schedule: testDoor('Scheduled backups', '/graph/backup/scheduled', {
      context: `of ${cluster}`,
    }),
  },
  ...overrides,
});

const DOORS = testView().backups.doors;

const renderSection = (backups: Partial<HubBackups>) =>
  renderHub(
    <BackupsSection
      technologyId="mongodb"
      backups={{ state: 'ok', rows: [], doors: DOORS, ...backups }}
      state={READY}
      messages={MESSAGES}
      onRetry={vi.fn()}
    />
  );

describe.skipIf(!hasPeakHub)('BackupsSection', () => {
  it('shows the last good backup of every cluster in a named table', () => {
    renderSection({
      rows: [
        row('orders', {
          lastGood: { at: minutesAgo(32 * 60), level: 'warning' },
          failedAt: minutesAgo(8 * 60),
          pitr: 'off',
        }),
        row('billing', { pitr: 'stopped' }),
        row('reports', {
          lastGood: null,
          pitr: 'notApplicable',
          nextRun: undefined,
        }),
      ],
    });

    const table = screen.getByRole('table', { name: 'Backups by cluster' });
    const orders = closestTo(
      within(table).getByRole('link', { name: 'orders' }),
      'tr'
    );
    const billing = closestTo(
      within(table).getByRole('link', { name: 'billing' }),
      'tr'
    );
    const reports = closestTo(
      within(table).getByRole('link', { name: 'reports' }),
      'tr'
    );

    expect(
      within(orders).getByRole('link', { name: 'failed 8h ago of orders' })
    ).toHaveAttribute('href', '/graph/backup/inventory');
    expect(orders).toHaveTextContent('last good 1d 8h ago');
    expect(
      within(billing).getByRole('link', { name: 'Stopped of billing' })
    ).toBeInTheDocument();
    expect(
      within(billing).getByRole('link', { name: '03:00 of billing' })
    ).toHaveAttribute('href', '/graph/backup/scheduled');
    expect(reports).toHaveTextContent('None yet');
    expect(within(reports).getAllByText('Not applicable')).toHaveLength(2);
  });

  it('opens MongoDB backups from the section and the dashboard from below the table', () => {
    renderSection({ rows: [row('billing')] });

    const section = screen.getByRole('region', { name: 'Backups' });

    expect(
      within(section).getByRole('link', { name: 'MongoDB backups' })
    ).toHaveAttribute('href', '/graph/backup/inventory');
    expect(
      within(section).getByRole('link', { name: 'Backup status dashboard' })
    ).toBeInTheDocument();
  });

  it('says how to start when nothing is set up, with the way forward for admins', () => {
    renderSection({
      state: 'notSetUp',
      hint: 'PBM is not configured on billing, orders',
      doors: {
        ...DOORS,
        setUp: testDoor('Set up a backup', '/graph/backup/new'),
      },
    });

    expect(
      screen.getByRole('heading', { level: 3, name: 'No MongoDB backups yet' })
    ).toBeInTheDocument();
    expect(
      screen.getByText(/PBM is not configured on billing, orders/)
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: 'Set up a backup' })
    ).toHaveAttribute('href', '/graph/backup/new');
  });

  it('stays visible with no services, without a way forward', () => {
    renderSection({ state: 'noServices' });

    expect(
      screen.getByText('Backups start once a MongoDB service is monitored.')
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: 'Set up a backup' })
    ).not.toBeInTheDocument();
  });

  it('says when backup management is off and where to turn it on', () => {
    renderSection({
      state: 'featureOff',
      doors: {
        ...DOORS,
        settings: testDoor(
          'Turn it on in Settings',
          '/settings/advanced-settings'
        ),
      },
    });

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Backup management is off.'
    );
    expect(
      screen.getByRole('link', { name: 'Turn it on in Settings' })
    ).toHaveAttribute('href', '/settings/advanced-settings');
  });
});
