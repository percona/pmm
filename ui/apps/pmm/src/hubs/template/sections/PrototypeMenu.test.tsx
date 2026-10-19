import { fireEvent, screen, within } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import type { HubParams } from '../hub.types';
import { PrototypeMenu } from './PrototypeMenu';

const LIVE: HubParams = {
  environment: null,
  scenario: 'live',
  ops: 'live',
  role: 'live',
};

const renderMenu = (params: Partial<HubParams> = {}) => {
  const setParams = vi.fn();

  renderHub(
    <PrototypeMenu params={{ ...LIVE, ...params }} setParams={setParams} />
  );

  return { setParams };
};

describe('PrototypeMenu', () => {
  it('names the state it shows, listing only what differs from live', () => {
    renderMenu({ scenario: 'wrong', ops: 'on', role: 'admin' });

    expect(
      screen.getByRole('button', {
        name: 'Prototype: Things wrong · Ops on · Admin',
      })
    ).toHaveAttribute('aria-haspopup', 'menu');
  });

  it('says Live while every switch follows the real data', () => {
    renderMenu();

    expect(
      screen.getByRole('button', { name: 'Prototype: Live' })
    ).toBeInTheDocument();
  });

  it('offers each switch as checked radio items', () => {
    renderMenu({ scenario: 'healthy', ops: 'off' });

    fireEvent.click(screen.getByRole('button', { name: /^Prototype/ }));

    const menu = screen.getByRole('menu');
    const radios = within(menu).getAllByRole('menuitemradio');

    expect(radios).toHaveLength(10);
    expect(
      within(menu).getByRole('menuitemradio', { name: 'Healthy' })
    ).toHaveAttribute('aria-checked', 'true');
    expect(
      within(menu).getByRole('menuitemradio', { name: 'Things wrong' })
    ).toHaveAttribute('aria-checked', 'false');
    expect(
      within(menu).getByRole('menuitemradio', { name: 'Off' })
    ).toHaveAttribute('aria-checked', 'true');
    expect(
      within(menu).getByRole('menuitemradio', { name: 'As signed in' })
    ).toHaveAttribute('aria-checked', 'true');
  });

  it.each<[string, Partial<HubParams>]>([
    ['Empty', { scenario: 'empty' }],
    ['On', { ops: 'on' }],
    ['Viewer', { role: 'viewer' }],
  ])('writes %s into the params and closes', (name, next) => {
    const { setParams } = renderMenu();

    fireEvent.click(screen.getByRole('button', { name: /^Prototype/ }));
    fireEvent.click(screen.getByRole('menuitemradio', { name }));

    expect(setParams).toHaveBeenCalledWith(next);
  });
});
