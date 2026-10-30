import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import type * as NavigationContext from 'contexts/navigation';
import type { NavItem } from 'types/navigation.types';
import type { User } from 'types/user.types';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { Messages } from '../search.messages';
import { getRecentKey, pushRecent } from '../search.recent';
import { openSearch } from '../search.store';
import {
  mockSearchApis,
  navTreeFor,
  renderWithSearchContexts,
  testItem,
} from '../search.testFixtures';
import { enableSearchForTest, hasPeakSearch } from '../testUtils';
import { SearchRuntime } from './SearchRuntime';

const nav = vi.hoisted(() => ({
  tree: [] as NavItem[],
  setNavOpen: vi.fn(),
}));

vi.mock('contexts/navigation', async (importOriginal) => ({
  ...(await importOriginal<typeof NavigationContext>()),
  useNavigation: () => ({
    navTree: nav.tree,
    navOpen: true,
    setNavOpen: nav.setNavOpen,
  }),
}));

const setup = async (
  user: User = TEST_USER_ADMIN,
  apis: Parameters<typeof mockSearchApis>[0] = {}
) => {
  nav.tree = navTreeFor(user);
  const mocks = mockSearchApis(apis);
  renderWithSearchContexts(<SearchRuntime />, { user });

  act(() => openSearch('sidebar'));

  const input = await screen.findByRole('combobox');

  return { ...mocks, input };
};

const type = (input: HTMLElement, value: string) =>
  fireEvent.change(input, { target: { value } });

// Group headings are aria-hidden, so the names come from the elements each group points at.
const groupNames = () =>
  screen
    .queryAllByRole('group')
    .map(
      (group) =>
        document.getElementById(group.getAttribute('aria-labelledby') ?? '')
          ?.textContent
    );

const optionsIn = (groupName: string) =>
  within(screen.getByRole('group', { name: groupName }))
    .getAllByRole('option')
    .map((option) => option.textContent ?? '');

describe.skipIf(!hasPeakSearch)('SearchPanel', () => {
  enableSearchForTest();

  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('starts with the starter pages, then actions and help', async () => {
    await setup();

    expect(groupNames()).toEqual(['Pages', 'Actions', 'Help']);
    expect(optionsIn('Pages')).toEqual([
      expect.stringMatching(/^Overview/),
      expect.stringMatching(/^Query analytics/),
      expect.stringMatching(/^All dashboards/),
      expect.stringMatching(/^Advisors/),
    ]);
    expect(optionsIn('Actions')[0]).toMatch(/^Add a service/);
    expect(optionsIn('Help')).toEqual([
      expect.stringMatching(/^Help/),
      expect.stringMatching(/^PMM documentation/),
    ]);
  });

  it('starts with Recent once pages were visited, showing live labels', async () => {
    pushRecent(
      getRecentKey(TEST_USER_ADMIN),
      testItem('Disk details', {
        id: 'dashboard:node-disk',
        kind: 'dashboard',
        to: '/graph/d/node-disk/disk-details',
      })
    );

    await setup();

    expect(groupNames()).toEqual(['Recent', 'Actions', 'Help']);
    expect(optionsIn('Recent')[0]).toMatch(/^Operating system disk/);
  });

  it('finds pages, dashboards and services for an admin', async () => {
    const { input } = await setup();

    type(input, 'mysql');

    await waitFor(() =>
      expect(groupNames()).toEqual(['Pages', 'Dashboards', 'Inventory'])
    );
    expect(
      screen.getByRole('option', { name: /^mysql-prod/ })
    ).toBeInTheDocument();
  });

  it('offers a viewer no inventory, settings or admin actions', async () => {
    const { input } = await setup(TEST_USER_VIEWER);

    type(input, 'mysql');
    await waitFor(() => expect(groupNames()).toEqual(['Pages', 'Dashboards']));

    type(input, 'data retention');
    await waitFor(() => expect(groupNames()).not.toContain('Settings'));

    type(input, 'add a service');
    await waitFor(() =>
      expect(
        screen.queryByRole('option', { name: /^Add a service/ })
      ).not.toBeInTheDocument()
    );
  });

  it('opens the first result on Enter', async () => {
    const { input } = await setup();

    type(input, 'qan');
    await screen.findByRole('option', { name: /^Query analytics/ });
    fireEvent.keyDown(input, { key: 'Enter' });

    await waitFor(() =>
      expect(screen.getByTestId('location')).toHaveTextContent(
        '/graph/d/pmm-qan/pmm-query-analytics'
      )
    );
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    );
  });

  it('explains an empty result', async () => {
    const { input } = await setup();

    type(input, 'xyzzy');

    expect(
      await screen.findByText('No results for “xyzzy”')
    ).toBeInTheDocument();
    expect(screen.getByText(Messages.noResultsHint)).toBeInTheDocument();
  });

  it('offers a retry when dashboards fail, and keeps the dialog open', async () => {
    const { input, searchGet } = await setup(TEST_USER_ADMIN, {
      dashboards: 'error',
    });

    type(input, 'disk');

    const retry = await screen.findByRole(
      'option',
      { name: /^Couldn't load dashboards/ },
      { timeout: 4000 }
    );
    const calls = searchGet.mock.calls.length;

    fireEvent.click(retry);

    await waitFor(() =>
      expect(searchGet.mock.calls.length).toBeGreaterThan(calls)
    );
    expect(
      screen.getByRole('dialog', { name: 'Search PMM' })
    ).toBeInTheDocument();
  });
});
