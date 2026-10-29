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
const STALE = 'Make the freshness indicator stale';
const OPERATIONS = 'Operations for MongoDB';
const toggle = (name: string) => within(panel()).getByRole('switch', { name });

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
    expect(toggle(STALE)).toBeChecked();
    expect(toggle(OPERATIONS)).not.toBeChecked();
  });

  it('writes each choice to the URL params', () => {
    const setParams = renderControls();

    fireEvent.click(trigger());
    fireEvent.click(
      within(group('Scenario')).getByRole('button', { name: 'Incomplete' })
    );
    fireEvent.click(toggle(OPERATIONS));
    fireEvent.click(toggle(STALE));

    expect(setParams.mock.calls).toEqual([
      [{ scenario: 'incomplete' }],
      [{ ops: 'off' }],
      [{ freshness: 'stale' }],
    ]);
  });

  it('restores the real 30-second refresh when stale is turned off', () => {
    const setParams = renderControls({ freshness: 'stale', refresh: '1m' });

    fireEvent.click(trigger());
    fireEvent.click(toggle(STALE));

    expect(setParams).toHaveBeenCalledWith({
      freshness: 'live',
      refresh: '30s',
    });
  });

  it('keeps Operations off for a viewer', () => {
    renderControls({}, { operations: false, admin: false });

    fireEvent.click(trigger());

    expect(toggle(OPERATIONS)).toBeDisabled();
    expect(toggle(OPERATIONS)).not.toBeChecked();
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
