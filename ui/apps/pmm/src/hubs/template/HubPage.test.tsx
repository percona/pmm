import { describe } from 'vitest';
import { fireEvent, screen, within } from '@testing-library/react';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { hasPeakHub, renderHub } from '../testUtils';
import { HubRoute } from './HubRoute';

type Scenario = 'empty' | 'healthy' | 'unhealthy';
type Ops = 'on' | 'off';
type Role = 'admin' | 'viewer';

const renderPage = ({
  scenario,
  ops,
  role,
  extensions = true,
}: {
  scenario: Scenario;
  ops: Ops;
  role: Role;
  extensions?: boolean;
}) =>
  renderHub(<HubRoute />, {
    path: `/mongodb?hubState=${scenario}&hubOps=${ops}&hubRole=${role}`,
    user: TEST_USER_ADMIN,
    settings: {
      extensionsEnabled: extensions,
      omEnabled: true,
      backupManagementEnabled: true,
      alertingEnabled: true,
    },
  });

const region = (name: string) => screen.queryByRole('region', { name });

const appTitle = (name: string) =>
  within(screen.getByRole('region', { name: 'Apps' })).queryByRole('heading', {
    level: 3,
    name,
  });

const MATRIX: Array<[Scenario, Ops, Role]> = (
  ['empty', 'healthy', 'unhealthy'] as Scenario[]
).flatMap((scenario) =>
  (['on', 'off'] as Ops[]).flatMap((ops) =>
    (['admin', 'viewer'] as Role[]).map((role): [Scenario, Ops, Role] => [
      scenario,
      ops,
      role,
    ])
  )
);

describe.skipIf(!hasPeakHub)('HubPage', () => {
  describe.each(MATRIX)('%s, Operations %s, %s', (scenario, ops, role) => {
    const admin = role === 'admin';
    const operations = admin && ops === 'on';

    it('has one h1 and a named section per h2', async () => {
      renderPage({ scenario, ops, role });

      expect(
        await screen.findByRole('heading', { level: 1, name: 'MongoDB' })
      ).toBeInTheDocument();
      expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);

      screen.getAllByRole('heading', { level: 2 }).forEach((heading) => {
        expect(
          screen.getByRole('region', { name: heading.textContent ?? '' })
        ).toContainElement(heading);
      });
    });

    it('reads as generally available: no Sample data or Preview label', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      expect(screen.queryByText('Sample data')).toBeNull();
      expect(screen.queryByText('Preview')).toBeNull();
    });

    it('shows Needs attention only while something needs attention, five rows at first', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });
      const attention = region('Needs attention');

      if (scenario === 'empty' || scenario === 'healthy') {
        expect(attention).toBeNull();
        return;
      }

      if (!attention) {
        throw new Error('No Needs attention section');
      }

      expect(within(attention).getAllByRole('listitem')).toHaveLength(5);
    });

    it('offers Add to admins and Install to admins with Operations', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      // Two ways in open a menu; one is a plain link.
      const add = screen.queryAllByRole(operations ? 'button' : 'link', {
        name: 'Add MongoDB',
      });

      expect(add.length > 0).toBe(admin);
      if (operations) {
        fireEvent.click(add[0]);
        expect(
          screen.getByRole('menuitem', {
            name: /^Install a new MongoDB cluster/,
          })
        ).toBeInTheDocument();
      }
    });

    it('shows the Operations card only to admins with Operations', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      expect(appTitle('Operations for MongoDB') != null).toBe(operations);
      expect(appTitle('MongoDB backups')).not.toBeNull();
    });

    it('leaves clusters and backups to their own pages, and keeps both Queries links', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      expect(region('Clusters')).toBeNull();
      expect(region('Backups')).toBeNull();

      const queries = screen.getByRole('region', { name: 'Queries' });

      expect(
        within(queries).getByRole('link', { name: /^Query analytics/ })
      ).toBeInTheDocument();
      expect(
        within(queries).getByRole('link', { name: /^Real-time queries/ })
      ).toBeInTheDocument();
    });

    it('never shows No data in the healthy color', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      screen
        .queryAllByTestId('hub-health')
        .filter((chip) => chip.textContent === 'No data')
        .forEach((chip) => {
          expect(chip).toHaveClass('MuiChip-colorDefault');
          expect(chip).not.toHaveClass('MuiChip-colorSuccess');
        });
    });
  });

  it.each<[boolean]>([[true], [false]])(
    'shows the Diagnostics card only with PMM Extensions on (%s)',
    async (extensions) => {
      renderPage({ scenario: 'healthy', ops: 'on', role: 'admin', extensions });

      await screen.findByRole('heading', { level: 1 });

      expect(appTitle('Diagnostics') != null).toBe(extensions);
    }
  );

  it.each<[Ops, number]>([
    ['off', 7],
    ['on', 7],
  ])(
    'folds the rest of what is wrong behind Show more, Operations %s',
    async (ops, more) => {
      renderPage({ scenario: 'unhealthy', ops, role: 'admin' });

      expect(
        await screen.findByRole('button', { name: `Show ${more} more` })
      ).toHaveAttribute('aria-expanded', 'false');
    }
  );

  it('opens every row of Needs attention from the banner’s Review link', async () => {
    renderPage({ scenario: 'unhealthy', ops: 'on', role: 'admin' });

    fireEvent.click(
      await screen.findByRole('link', { name: 'Review 39 issues' })
    );

    expect(
      await screen.findByRole('button', { name: 'Show fewer' })
    ).toHaveAttribute('aria-expanded', 'true');
  });

  it('lays the sections out in reading order: Needs attention, Queries, Apps, Dashboards', async () => {
    renderPage({ scenario: 'unhealthy', ops: 'on', role: 'admin' });

    await screen.findByRole('heading', { level: 1 });

    expect(
      screen
        .getAllByRole('heading', { level: 2 })
        .map((heading) => heading.textContent)
    ).toEqual(['Needs attention', 'Queries', 'Apps', 'Dashboards']);
  });

  it('survives a malformed hash in a pasted link', async () => {
    renderHub(<HubRoute />, { path: '/mongodb?hubState=healthy#100%' });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'MongoDB' })
    ).toBeInTheDocument();
  });

  it('keeps a filter that matches nothing visible', async () => {
    renderHub(<HubRoute />, {
      path: '/mongodb?environment=qa&hubState=healthy',
      user: TEST_USER_ADMIN,
      settings: { backupManagementEnabled: true },
    });

    expect(await screen.findByTestId('hub-status-banner')).toHaveTextContent(
      'No MongoDB in this environment'
    );
    expect(
      screen.getByRole('combobox', { name: 'Environment' })
    ).toHaveTextContent('qa');
    expect(region('Needs attention')).toBeNull();
  });
});
