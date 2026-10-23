import { describe } from 'vitest';
import { createTheme } from '@mui/material/styles';
import { baseThemeOptions, PerconaMoIcon } from '@percona/peak-ui';
import { act, fireEvent, screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import { testView } from '../hub.testFixtures';
import type { HubFreshness, HubMessages, HubViewModel } from '../hub.types';
import { HUB_TURN_MS } from './HubFreshness';
import { HubHeader } from './HubHeader';

const MESSAGES = {
  name: 'MongoDB',
  unit: { cluster: ['cluster', 'clusters'] },
} as HubMessages;

const FRESH: HubFreshness = {
  state: 'live',
  at: Date.now() - 20_000,
  interval: '30s',
  refreshing: false,
};

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

  it('reads Live as its interval on the RTA selector, with no ticking age, and refreshes on demand', () => {
    const { refetch } = renderHeader();
    const widget = screen.getByTestId('hub-freshness');

    expect(screen.getByTestId('hub-auto-refresh')).toHaveTextContent(
      /^Auto-refresh: 30s$/
    );
    expect(widget).not.toHaveTextContent(/ago/);

    fireEvent.click(screen.getByRole('button', { name: 'Refresh now' }));

    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it('turns Refresh while a refresh runs, disabled until a whole turn ends', () => {
    vi.useFakeTimers();
    try {
      const handlers = { setParams: vi.fn(), refetch: vi.fn() };
      const { rerender } = renderHub(header({}, FRESH, handlers));
      const button = screen.getByRole('button', { name: 'Refresh now' });
      const icon = () => button.querySelector('svg') as SVGElement;

      expect(button).not.toHaveAttribute('aria-disabled');

      // An automatic refresh in flight, longer than a turn.
      rerender(header({}, { ...FRESH, refreshing: true }, handlers));
      act(() => vi.advanceTimersByTime(HUB_TURN_MS * 2));

      expect(icon()).toHaveAttribute('data-turning', 'true');
      expect(button).toHaveAttribute('aria-disabled', 'true');

      fireEvent.click(button);

      expect(handlers.refetch).not.toHaveBeenCalled();

      // It lands partway through the third turn, which still completes.
      act(() => vi.advanceTimersByTime(HUB_TURN_MS / 2));
      rerender(header({}, { ...FRESH, at: Date.now() }, handlers));
      act(() => vi.advanceTimersByTime(HUB_TURN_MS / 2 - 1));

      expect(button).toHaveAttribute('aria-disabled', 'true');

      act(() => vi.advanceTimersByTime(1));

      expect(button).not.toHaveAttribute('aria-disabled');
      expect(icon()).not.toHaveAttribute('data-turning');
    } finally {
      vi.useRealTimers();
    }
  });

  it('turns Refresh once for a manual refresh, however quick', () => {
    vi.useFakeTimers();
    try {
      const { refetch } = renderHeader();
      const button = screen.getByRole('button', { name: 'Refresh now' });

      fireEvent.click(button);

      expect(refetch).toHaveBeenCalledTimes(1);
      expect(button).toHaveAttribute('aria-disabled', 'true');

      act(() => vi.advanceTimersByTime(HUB_TURN_MS));

      expect(button).not.toHaveAttribute('aria-disabled');
    } finally {
      vi.useRealTimers();
    }
  });

  it('shows the selector in Peak’s neutral Base theme, right beside Refresh, with no tooltip', () => {
    renderHeader();
    const selector = screen.getByTestId('hub-auto-refresh');
    const refresh = screen.getByRole('button', { name: 'Refresh now' });

    expect(selector).toHaveClass('MuiButton-sizeMedium');
    expect(
      getComputedStyle(selector).getPropertyValue('--variant-textColor')
    ).toBe(createTheme(baseThemeOptions('light')).palette.primary.main);
    expect(refresh).toHaveClass('MuiIconButton-sizeMedium');
    expect(refresh).not.toHaveClass('MuiIconButton-colorSecondary');
    expect(screen.getByTestId('hub-freshness')).toHaveStyle({ gap: '0px' });

    fireEvent.mouseOver(selector);

    expect(screen.queryByRole('tooltip')).toBeNull();
  });

  it('picks the interval from Grafana’s set, Off first', () => {
    const { setParams } = renderHeader();

    fireEvent.click(screen.getByTestId('hub-auto-refresh'));

    expect(
      within(screen.getByTestId('text-select-menu'))
        .getAllByRole('menuitem')
        .map((item) => item.textContent)
    ).toEqual([
      'Off',
      '5s',
      '10s',
      '30s',
      '1m',
      '5m',
      '15m',
      '30m',
      '1h',
      '2h',
      '1d',
    ]);

    fireEvent.click(screen.getByTestId('text-select-option-1m'));

    expect(setParams).toHaveBeenCalledWith({ refresh: '1m' });
  });

  it('reads OFF with auto-refresh off, keeping the selector and Refresh', () => {
    renderHeader({}, { ...FRESH, state: 'off', interval: 'off' });

    expect(screen.getByTestId('hub-auto-refresh')).toHaveTextContent(
      /^Auto-refresh: OFF$/
    );
    expect(
      screen.getByRole('button', { name: 'Refresh now' })
    ).toBeInTheDocument();
  });

  it('warns when the data is stale, with how long ago, Retry, and once to assistive tech', () => {
    const { refetch } = renderHeader(
      {},
      { ...FRESH, state: 'stale', at: Date.now() - 4 * 60_000 }
    );

    const line = screen.getByText('Data paused, last update 4m ago');

    // Peak's small State indicator: a 20px warning glyph on a body1 line.
    expect(line.closest('[data-state="degraded"]')).not.toBeNull();
    expect(getComputedStyle(line).fontSize).toBe('1rem');
    expect(
      screen
        .getByTestId('hub-freshness')
        .querySelector('[data-glyph="warning"]')
    ).toHaveStyle({ width: '20px' });
    expect(screen.getByRole('button', { name: 'Retry' })).toHaveClass(
      'MuiButton-sizeMedium'
    );
    expect(screen.queryByTestId('hub-auto-refresh')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Refresh now' })).toBeNull();
    expect(liveRegion()).toHaveTextContent('Data paused.');

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

    expect(refetch).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('button', { name: 'Retry' })).toHaveAttribute(
      'aria-disabled',
      'true'
    );
  });

  it('tells assistive tech when a refresh it asked for has landed', () => {
    const handlers = { setParams: vi.fn(), refetch: vi.fn() };
    const before = Date.now() - 20_000;
    const { rerender } = renderHub(
      header({}, { ...FRESH, at: before }, handlers)
    );

    expect(liveRegion()).toHaveTextContent('');

    fireEvent.click(screen.getByRole('button', { name: 'Refresh now' }));
    rerender(header({}, { ...FRESH, at: before }, handlers));

    expect(handlers.refetch).toHaveBeenCalledTimes(1);
    expect(liveRegion()).toHaveTextContent('');

    rerender(header({}, { ...FRESH, at: Date.now() }, handlers));

    expect(liveRegion()).toHaveTextContent('Updated just now');
  });
});
