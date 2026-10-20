import { describe } from 'vitest';
import { fireEvent, screen, within } from '@testing-library/react';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { hasPeakHub, renderHub } from '../testUtils';
import { HubRoute } from './HubRoute';

type Scenario = 'empty' | 'healthy' | 'wrong';
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
  ['empty', 'healthy', 'wrong'] as Scenario[]
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

    it('marks the sample data', async () => {
      renderPage({ scenario, ops, role });

      expect(await screen.findByText('Sample data')).toBeInTheDocument();
    });

    it('shows Needs attention only when there is an estate, and its list only when wrong', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });
      const attention = region('Needs attention');

      if (scenario === 'empty') {
        expect(attention).toBeNull();
        return;
      }

      if (!attention) {
        throw new Error('No Needs attention section');
      }

      const section = within(attention);

      if (scenario === 'healthy') {
        expect(
          section.getByText(/^Nothing needs attention/)
        ).toBeInTheDocument();
        expect(section.queryByRole('list')).toBeNull();
      } else {
        expect(section.getAllByRole('listitem')).toHaveLength(5);
        expect(section.queryByText(/^Nothing needs attention/)).toBeNull();
      }
    });

    it('offers Add to admins and Install to admins with Operations', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      expect(
        screen.queryAllByRole('link', { name: 'Add MongoDB service' }).length >
          0
      ).toBe(admin);
      expect(
        screen.queryAllByRole('link', { name: /^Install MongoDB/ }).length > 0
      ).toBe(operations);
    });

    it('shows the Operations card only to admins with Operations', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      expect(appTitle('Operations for MongoDB') != null).toBe(operations);
      expect(appTitle('MongoDB backups')).not.toBeNull();
    });

    it('keeps Backups and both Queries links', async () => {
      renderPage({ scenario, ops, role });

      await screen.findByRole('heading', { level: 1 });

      expect(region('Backups')).not.toBeNull();

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
    ['off', 8],
    ['on', 9],
  ])(
    'folds the rest of what is wrong behind Show more, Operations %s',
    async (ops, more) => {
      renderPage({ scenario: 'wrong', ops, role: 'admin' });

      expect(
        await screen.findByRole('button', { name: `Show ${more} more` })
      ).toHaveAttribute('aria-expanded', 'false');
    }
  );

  it('opens the problem path of the tree and leaves healthy branches folded', async () => {
    renderPage({ scenario: 'healthy', ops: 'off', role: 'admin' });

    const table = await screen.findByRole('table', {
      name: 'MongoDB clusters',
    });

    expect(
      within(table).queryAllByRole('button', { name: /^Collapse / })
    ).toHaveLength(0);
  });

  it('shows how to start in the Clusters section of an empty estate', async () => {
    renderPage({ scenario: 'empty', ops: 'on', role: 'viewer' });

    const clusters = await screen.findByRole('region', { name: 'Clusters' });

    expect(
      within(clusters).getByRole('heading', {
        level: 3,
        name: 'Start monitoring MongoDB',
      })
    ).toBeInTheDocument();
    expect(
      within(clusters).getByText(
        'Ask an administrator to add a MongoDB service.'
      )
    ).toBeInTheDocument();
    expect(within(clusters).queryByRole('link')).toBeNull();
  });

  it('survives a malformed hash in a pasted link', async () => {
    renderHub(<HubRoute />, { path: '/mongodb?hubState=healthy#100%' });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'MongoDB' })
    ).toBeInTheDocument();
  });

  it('keeps a filter that matches nothing visible and undoable', async () => {
    renderHub(<HubRoute />, {
      path: '/mongodb?environment=qa&hubState=healthy',
      user: TEST_USER_ADMIN,
      settings: { backupManagementEnabled: true },
    });

    expect(await screen.findByTestId('hub-status-line')).toHaveTextContent(
      'No MongoDB in this environment'
    );
    expect(
      screen.getByRole('combobox', { name: 'Environment' })
    ).toHaveTextContent('qa');
    expect(region('Needs attention')).toBeNull();
    expect(
      screen.queryByRole('heading', { name: 'Start monitoring MongoDB' })
    ).toBeNull();
    expect(
      within(screen.getByRole('region', { name: 'Backups' })).getByText(
        'No MongoDB service is in this environment.'
      )
    ).toBeInTheDocument();

    const clusters = screen.getByRole('region', { name: 'Clusters' });

    expect(
      within(clusters).getByRole('heading', {
        level: 3,
        name: 'No matching clusters',
      })
    ).toBeInTheDocument();

    fireEvent.click(
      within(clusters).getByRole('button', { name: 'Show all environments' })
    );

    expect(screen.getByTestId('hub-location')).toHaveTextContent(
      /^\/mongodb\?hubState=healthy$/
    );
  });
});
