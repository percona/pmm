import { describe } from 'vitest';
import { fireEvent, screen, within } from '@testing-library/react';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { hasPeakHub, renderHub } from '../testUtils';
import { HubRoute } from './HubRoute';

type Scenario = 'empty' | 'healthy' | 'incomplete' | 'unhealthy';
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

const tiles = () =>
  within(screen.getByRole('navigation', { name: 'Shortcuts' }))
    .getAllByRole('link')
    .map((link) => link.textContent);

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

    it('shows the same tiles in every state, Fleet only to admins with Operations', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      expect(tiles()).toEqual([
        'Clusters',
        'Instances overview',
        'Query analytics',
        'Real-time queries',
        'MongoDB backups',
        ...(operations ? ['Fleet'] : []),
      ]);
    });

    it('leaves clusters, backups and apps to their own pages, and the ways into queries to the tiles', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      expect(region('Clusters')).toBeNull();
      expect(region('Backups')).toBeNull();
      expect(region('Apps')).toBeNull();
      expect(
        within(screen.getByRole('region', { name: 'Queries' })).queryByRole(
          'link',
          { name: /^(Query analytics|Real-time queries)/ }
        )
      ).toBeNull();
      expect(
        within(screen.getByRole('region', { name: 'Dashboards' })).getByRole(
          'link',
          { name: 'All apps' }
        )
      ).toHaveAttribute('href', '/prototype/all-apps');
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

  it('lays the page out in reading order: tiles, then Needs attention, Recommendations, Queries, Dashboards', async () => {
    renderPage({ scenario: 'unhealthy', ops: 'on', role: 'admin' });

    await screen.findByRole('heading', { level: 1 });

    expect(
      screen
        .getAllByRole('heading', { level: 2 })
        .map((heading) => heading.textContent)
    ).toEqual(['Needs attention', 'Recommendations', 'Queries', 'Dashboards']);
    expect(
      screen
        .getByRole('navigation', { name: 'Shortcuts' })
        .compareDocumentPosition(
          screen.getByRole('heading', { level: 2, name: 'Needs attention' })
        )
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it('lays a Work in progress line over the page that clicks go through', async () => {
    renderPage({ scenario: 'healthy', ops: 'on', role: 'admin' });

    const line = await screen.findByText('Work in progress');
    // The line, the part held in the window, the layer over the page.
    const layer = line.parentElement?.parentElement as HTMLElement;

    expect(layer).toHaveStyle({ position: 'absolute', pointerEvents: 'none' });
    expect(
      line.compareDocumentPosition(screen.getByRole('heading', { level: 1 }))
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it.each<[Scenario, number, string]>([
    ['unhealthy', 5, 'Show 1 more'],
    ['incomplete', 5, 'Show 1 more'],
    ['healthy', 6, ''],
  ])(
    'shows five recommendations beside Needs attention, six alone: %s shows %i',
    async (scenario, rows, more) => {
      renderPage({ scenario, ops: 'on', role: 'admin' });

      const section = await screen.findByRole('region', {
        name: 'Recommendations',
      });

      expect(within(section).getAllByRole('listitem')).toHaveLength(rows);
      expect(
        within(section).queryByRole('button', { name: /^Show / })
          ?.textContent ?? ''
      ).toBe(more);
      expect(region('Needs attention') === null).toBe(scenario === 'healthy');
    }
  );

  it('matches the folded lists’ heights, and lets an open one grow alone', async () => {
    renderPage({ scenario: 'unhealthy', ops: 'on', role: 'admin' });

    const attention = await screen.findByRole('region', {
      name: 'Needs attention',
    });
    // The row both sections sit in: section, cell, row.
    const row = attention.parentElement?.parentElement as HTMLElement;

    expect(row).toHaveStyle({ alignItems: 'stretch' });
    fireEvent.click(
      within(attention).getByRole('button', { name: 'Show 7 more' })
    );
    expect(row).toHaveStyle({ alignItems: 'start' });
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
