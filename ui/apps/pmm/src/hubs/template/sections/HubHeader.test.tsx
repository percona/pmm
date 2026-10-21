import { describe } from 'vitest';
import { createTheme } from '@mui/material/styles';
import { baseThemeOptions, PerconaMoIcon } from '@percona/peak-ui';
import { fireEvent, screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import { testView } from '../hub.testFixtures';
import type { HubFreshness, HubMessages, HubViewModel } from '../hub.types';
import { HubHeader } from './HubHeader';

const MESSAGES = {
  name: 'MongoDB',
  unit: { cluster: ['cluster', 'clusters'] },
} as HubMessages;

const FRESH: HubFreshness = { at: Date.now() - 20_000, stale: false };

const header = (
  view: Partial<HubViewModel> = {},
  freshness: HubFreshness = FRESH,
  handlers = { setParams: vi.fn(), refetch: vi.fn() }
) => (
  <HubHeader
    view={testView(view)}
    icon={PerconaMoIcon}
    messages={MESSAGES}
    setParams={handlers.setParams}
    freshness={freshness}
    refetch={handlers.refetch}
  />
);

const renderHeader = (
  view: Partial<HubViewModel> = {},
  freshness: HubFreshness = FRESH
) => {
  const handlers = { setParams: vi.fn(), refetch: vi.fn() };

  renderHub(header(view, freshness, handlers));

  return handlers;
};

// renderHub's location probe is an <output>, which is a status too.
const liveRegion = () =>
  screen
    .getAllByRole('status')
    .filter((element) => element.tagName !== 'OUTPUT')[0];

describe.skipIf(!hasPeakHub)('HubHeader', () => {
  it('is the page’s one h1, in the h3 size, and leaves the status line to the page', () => {
    renderHeader();

    const [heading] = screen.getAllByRole('heading', { level: 1 });

    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);
    expect(heading).toHaveAccessibleName('MongoDB');
    expect(heading).toHaveClass('MuiTypography-h3');
    expect(screen.queryByTestId('hub-status-banner')).not.toBeInTheDocument();
  });

  it('draws only Add in Peak’s neutral Base theme', () => {
    renderHeader();

    const { primary } = createTheme(baseThemeOptions('light')).palette;

    // MUI draws a text button's colour through this variable.
    expect(
      getComputedStyle(
        screen.getByRole('button', { name: 'Add MongoDB' })
      ).getPropertyValue('--variant-textColor')
    ).toBe(primary.main);
  });

  it('shows no Sample data or Preview label', () => {
    renderHeader({ scenario: 'healthy' });

    expect(screen.queryByText('Sample data')).not.toBeInTheDocument();
    expect(screen.queryByText(/preview/i)).not.toBeInTheDocument();
  });

  it('reads Add on screen and Add MongoDB to assistive tech, and offers the ways in as a menu', () => {
    renderHeader();

    const add = screen.getByRole('button', { name: 'Add MongoDB' });

    expect(add).toHaveTextContent(/^Add/);
    expect(add).toHaveAttribute('aria-haspopup', 'menu');
    fireEvent.click(add);

    const menu = screen.getByRole('menu', { name: 'Add MongoDB' });
    const monitor = within(menu).getByRole('menuitem', {
      name: /Monitor an existing MongoDB service/,
    });

    expect(monitor).toHaveAttribute('href', '/graph/add-instance/mongodb');
    expect(monitor).toHaveTextContent(/^Monitor an existing MongoDB service$/);
    expect(
      within(menu).getByRole('menuitem', {
        name: /Install a new MongoDB cluster/,
      })
    ).toHaveAttribute('href', '/operations/nodes?filter=unmonitored');
  });

  it('links straight to the one way in when there is only one', () => {
    const [monitor] = testView().actions.add;

    renderHeader({ actions: { add: [monitor], more: [] } });

    expect(screen.getByRole('link', { name: 'Add MongoDB' })).toHaveAttribute(
      'href',
      '/graph/add-instance/mongodb'
    );
  });

  it('lists the admin pages and the docs under More actions, the docs in a new tab', () => {
    renderHeader();

    fireEvent.click(screen.getByRole('button', { name: 'More actions' }));

    const menu = screen.getByRole('menu', { name: 'More actions' });
    const docs = within(menu).getByRole('menuitem', {
      name: 'Documentation (opens in a new tab)',
    });

    expect(
      within(menu).getByRole('menuitem', { name: 'Manage services' })
    ).toHaveAttribute('href', '/graph/inventory/services');
    expect(docs).toHaveAttribute('target', '_blank');
    expect(docs).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('shows no actions when there are none', () => {
    renderHeader({ actions: { add: [], more: [] } });

    expect(screen.queryByRole('button', { name: 'Add MongoDB' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'More actions' })).toBeNull();
  });

  it('names the environment filter and filters through the params', () => {
    const { setParams } = renderHeader();

    fireEvent.mouseDown(screen.getByRole('combobox', { name: /^Environment/ }));
    fireEvent.click(
      within(screen.getByRole('listbox')).getByRole('option', { name: 'prod' })
    );

    expect(setParams).toHaveBeenCalledWith({ environment: 'prod' });
  });

  it('clears the environment with All environments', () => {
    const { setParams } = renderHeader({
      environment: { ...testView().environment, selected: 'prod' },
    });

    fireEvent.mouseDown(screen.getByRole('combobox', { name: /^Environment/ }));
    fireEvent.click(screen.getByRole('option', { name: 'All environments' }));

    expect(setParams).toHaveBeenCalledWith({ environment: null });
  });

  it('hides the environment select with a single environment', () => {
    renderHeader({
      environment: {
        selected: null,
        options: [{ value: 'prod', label: 'prod' }],
        matchesNothing: false,
      },
    });

    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
  });

  it('says how fresh the data is and refreshes every source', () => {
    const { refetch } = renderHeader();

    expect(screen.getByText(/^Updated 2\ds ago$/)).toBeInTheDocument();
    expect(screen.queryByRole('img', { name: 'Warning' })).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));

    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it('warns when the data is stale, in words and on focus, and says so to assistive tech', async () => {
    renderHeader({}, { at: Date.now() - 32 * 60_000, stale: true });

    const age = screen.getByText('Updated 32m ago');

    expect(screen.getByRole('img', { name: 'Warning' })).toBeInTheDocument();
    expect(age).toHaveAttribute('tabindex', '0');
    expect(liveRegion()).toHaveTextContent(
      'Data looks stale. Refresh, or check your connection.'
    );

    fireEvent.mouseOver(age);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Data looks stale. Refresh, or check your connection.'
    );
  });

  it('tells assistive tech when a refresh it asked for has landed', () => {
    const handlers = { setParams: vi.fn(), refetch: vi.fn() };
    const before = Date.now() - 20_000;
    const { rerender } = renderHub(
      header({}, { at: before, stale: false }, handlers)
    );

    expect(liveRegion()).toHaveTextContent('');

    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    rerender(header({}, { at: before, stale: false }, handlers));

    expect(handlers.refetch).toHaveBeenCalledTimes(1);
    expect(liveRegion()).toHaveTextContent('');

    rerender(header({}, { at: Date.now(), stale: false }, handlers));

    expect(liveRegion()).toHaveTextContent('Updated just now');
  });
});
