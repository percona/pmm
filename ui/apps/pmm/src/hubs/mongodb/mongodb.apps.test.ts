import { ExtensionIcon } from 'lib/peak-ui-pending-icons';
import { healthyOperations } from './fixtures/estate';
import { type AppsInput, deriveApps } from './mongodb.apps';
import { NOW } from './mongodb.testUtils';

const input = (overrides: Partial<AppsInput> = {}): AppsInput => ({
  capabilities: { operations: true, extensions: true },
  emptyEstate: false,
  operations: healthyOperations(NOW),
  backups: {
    featureOff: false,
    notSetUp: false,
    schedules: 2,
    failedRecently: 0,
    lastGoodBackup: {
      at: new Date(NOW - 6 * 3600_000).toISOString(),
      cluster: 'billing',
    },
  },
  now: NOW,
  ...overrides,
});

describe('deriveApps', () => {
  it('shows Operations, MongoDB backups and Diagnostics in that order', () => {
    const apps = deriveApps(input());

    expect(apps.map((app) => app.name)).toEqual([
      'Operations for MongoDB',
      'MongoDB backups',
      'Diagnostics',
    ]);
    expect(apps.every((app) => app.icon === ExtensionIcon)).toBe(true);
  });

  it('marks Operations as a preview and links its pages', () => {
    const [operations] = deriveApps(input());

    expect(operations).toMatchObject({
      preview: 'developer',
      summary: {
        text: '15 of 15 services up · last scan 4m ago',
        doors: [
          { label: '15 of 15 services up', to: '/operations?tab=services' },
          {
            label: 'last scan 4m ago',
            to: '/operations/automations?tab=scans',
          },
        ],
      },
      pages: [
        { label: 'Fleet', to: '/operations' },
        { label: 'Nodes', to: '/operations/nodes' },
        { label: 'Automations', to: '/operations/automations' },
      ],
    });
  });

  it('points an empty estate at Nodes', () => {
    expect(deriveApps(input({ emptyEstate: true }))[0].summary).toEqual({
      text: 'No MongoDB yet. Install one from Nodes.',
    });
  });

  it('hides Operations without the capability and Diagnostics without PMM Extensions', () => {
    expect(
      deriveApps(
        input({ capabilities: { operations: false, extensions: false } })
      ).map((app) => app.id)
    ).toEqual(['backups']);
  });

  it('summarises backups by schedules and the last good one, or by failures', () => {
    const backupsOf = (backups: Partial<AppsInput['backups']>) =>
      deriveApps(input({ backups: { ...input().backups, ...backups } }))[1]
        .summary?.text;

    expect(backupsOf({})).toBe('2 schedules · last good backup 6h ago');
    expect(backupsOf({ failedRecently: 1 })).toBe(
      '2 schedules · 1 failed in the last 24h'
    );
    expect(backupsOf({ lastGoodBackup: 'none', schedules: 1 })).toBe(
      '1 schedule · no good backup yet'
    );
    expect(backupsOf({ featureOff: true })).toBe('Backup management is off');
    expect(backupsOf({ notSetUp: true })).toBe('No backups set up yet');
    expect(backupsOf({ lastGoodBackup: 'unknown', schedules: 0 })).toBe(
      'Backup status unknown'
    );
  });

  it('makes each backups figure a door', () => {
    expect(
      deriveApps(
        input({ backups: { ...input().backups, failedRecently: 1 } })
      )[1].summary?.doors
    ).toEqual([
      { label: '2 schedules', to: '/graph/backup/scheduled' },
      { label: '1 failed in the last 24h', to: '/graph/backup/inventory' },
    ]);
    expect(deriveApps(input())[1].summary?.doors?.[1]).toEqual({
      label: 'last good backup 6h ago',
      to: '/graph/backup/inventory',
    });
  });

  it('links the MongoDB backups pages and Diagnostics', () => {
    const apps = deriveApps(input());

    expect(apps[1].pages.map((page) => page.to)).toEqual([
      '/graph/backup/inventory',
      '/graph/backup/scheduled',
      '/graph/backup/restore',
      '/graph/backup/locations',
    ]);
    expect(apps[2]).toMatchObject({
      summary: {
        text: 'Diagnostic checks for MongoDB replication and performance',
      },
      pages: [{ label: 'Open Diagnostics', to: '/extensions/atw' }],
    });
  });
});
