import { act, fireEvent, screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import { DEFAULT_HUB_PARAMS } from '../hub.params';
import type { HubParams } from '../hub.types';
import { DemoControls } from './DemoControls';

const renderControls = (
  params: Partial<HubParams> = {},
  { operations = true, admin = true } = {}
) => {
  const setParams = vi.fn();

  renderHub(
    <>
      <input aria-label="Search" />
      <DemoControls
        name="MongoDB"
        params={{ ...DEFAULT_HUB_PARAMS, ...params }}
        setParams={setParams}
        operations={operations}
        admin={admin}
      />
    </>
  );

  return setParams;
};

const trigger = () => screen.getByRole('button', { name: 'Demo controls' });
const panel = () => screen.getByRole('dialog', { name: 'Demo controls' });
const group = (name: string) => within(panel()).getByRole('group', { name });
const pressed = (name: string) =>
  within(group(name))
    .getAllByRole('button')
    .filter((button) => button.getAttribute('aria-pressed') === 'true')
    .map((button) => button.textContent);

describe.skipIf(!hasPeakHub)('DemoControls', () => {
  it('opens a non-modal panel above its button, focused, with the URL’s state', () => {
    renderControls(
      { scenario: 'unhealthy', freshness: 'stale' },
      { operations: false }
    );

    expect(screen.queryByRole('dialog')).toBeNull();
    expect(trigger()).toHaveAttribute('aria-keyshortcuts', 'Shift+D');

    fireEvent.click(trigger());

    expect(trigger()).toHaveAttribute('aria-expanded', 'true');
    expect(panel()).toHaveFocus();
    expect(pressed('Scenario')).toEqual(['Unhealthy']);
    expect(pressed('Freshness')).toEqual(['Stale']);
    expect(pressed('Operations for MongoDB')).toEqual(['Off']);
  });

  it('writes each choice to the URL params', () => {
    const setParams = renderControls();

    fireEvent.click(trigger());
    fireEvent.click(
      within(group('Scenario')).getByRole('button', { name: 'Incomplete' })
    );
    fireEvent.click(
      within(group('Operations for MongoDB')).getByRole('button', {
        name: 'Off',
      })
    );
    fireEvent.click(
      within(group('Freshness')).getByRole('button', { name: 'Stale' })
    );

    expect(setParams.mock.calls).toEqual([
      [{ scenario: 'incomplete' }],
      [{ ops: 'off' }],
      [{ freshness: 'stale' }],
    ]);
  });

  it('offers only Live and Stale, since the selector already turns auto-refresh off', () => {
    renderControls();

    fireEvent.click(trigger());

    expect(
      within(group('Freshness'))
        .getAllByRole('button')
        .map((button) => button.textContent)
    ).toEqual(['Live', 'Stale']);
  });

  it('restores the real 30-second refresh with Live', () => {
    const setParams = renderControls({ freshness: 'stale', refresh: '1m' });

    fireEvent.click(trigger());

    expect(pressed('Freshness')).toEqual(['Stale']);

    fireEvent.click(
      within(group('Freshness')).getByRole('button', { name: 'Live' })
    );

    expect(setParams).toHaveBeenCalledWith({
      freshness: 'live',
      refresh: '30s',
    });
  });

  it('keeps Operations off for a viewer', () => {
    renderControls({}, { operations: false, admin: false });

    fireEvent.click(trigger());

    expect(
      within(group('Operations for MongoDB')).getByRole('button', {
        name: 'On',
      })
    ).toBeDisabled();
  });

  it('closes with Escape, returning focus, and with a click outside', async () => {
    renderControls();

    fireEvent.click(trigger());
    fireEvent.keyDown(panel(), { key: 'Escape' });

    expect(screen.queryByRole('dialog')).toBeNull();
    expect(trigger()).toHaveFocus();

    fireEvent.click(trigger());
    // MUI's ClickAwayListener ignores the click that opened it, until the next tick.
    await act(() => new Promise((resolve) => setTimeout(resolve, 0)));
    fireEvent.click(document.body);

    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('toggles with Shift+D, except while typing', () => {
    renderControls();

    fireEvent.keyDown(window, { key: 'D', code: 'KeyD', shiftKey: true });

    expect(panel()).toHaveFocus();

    fireEvent.keyDown(window, { key: 'D', code: 'KeyD', shiftKey: true });

    expect(screen.queryByRole('dialog')).toBeNull();
    expect(trigger()).toHaveFocus();

    const search = screen.getByRole('textbox', { name: 'Search' });

    search.focus();
    fireEvent.keyDown(search, { key: 'D', code: 'KeyD', shiftKey: true });
    fireEvent.keyDown(window, {
      key: 'D',
      code: 'KeyD',
      shiftKey: true,
      metaKey: true,
    });

    expect(screen.queryByRole('dialog')).toBeNull();
  });
});
