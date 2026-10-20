import { fireEvent, screen, within } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import type { HubParams } from '../hub.types';
import { PrototypeMenu, type PrototypeMenuProps } from './PrototypeMenu';

const LIVE: HubParams = {
  environment: null,
  patches: null,
  connection: 'live',
  scenario: 'live',
  ops: 'live',
  role: 'live',
};

const renderMenu = (
  params: Partial<HubParams> = {},
  {
    role = 'admin',
    operations = params.ops === 'on',
    access = { admin: true, operations: true },
  }: Partial<Pick<PrototypeMenuProps, 'role' | 'operations' | 'access'>> = {}
) => {
  const setParams = vi.fn();

  renderHub(
    <PrototypeMenu
      params={{ ...LIVE, ...params }}
      setParams={setParams}
      role={role}
      operations={operations}
      access={access}
    />
  );

  return { setParams };
};

const open = () => {
  fireEvent.click(screen.getByRole('button', { name: /^Prototype/ }));

  return screen.getByRole('menu');
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

    expect(radios).toHaveLength(12);
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

  it('separates its four radio sets and names each item’s set', () => {
    renderMenu({ scenario: 'wrong' });

    const menu = open();

    expect(within(menu).getAllByRole('separator')).toHaveLength(3);

    const wrong = within(menu).getByRole('menuitemradio', {
      name: 'Things wrong',
    });

    expect(wrong).toHaveAccessibleDescription('Data');
    expect(wrong).toHaveAttribute('aria-posinset', '4');
    expect(wrong).toHaveAttribute('aria-setsize', '4');
    expect(
      within(menu).getByRole('menuitemradio', { name: 'As signed in' })
    ).toHaveAccessibleDescription('Signed in as');
  });

  it('loses the connection only on sample data, and says so in its name', () => {
    const { setParams } = renderMenu({ scenario: 'wrong' });
    const menu = open();

    fireEvent.click(within(menu).getByRole('menuitemradio', { name: 'Lost' }));
    expect(setParams).toHaveBeenCalledWith({ connection: 'lost' });
  });

  it('names a lost connection on the button, and keeps it from live data', () => {
    renderMenu({ scenario: 'wrong', connection: 'lost' });

    expect(
      screen.getByRole('button', {
        name: 'Prototype: Things wrong · Connection lost',
      })
    ).toBeInTheDocument();
  });

  it('cannot lose the connection of live data', () => {
    renderMenu({ connection: 'lost' });

    const menu = open();

    expect(
      within(menu).getByRole('menuitemradio', { name: /^Lost/ })
    ).toHaveAttribute('aria-disabled', 'true');
    expect(
      within(menu).getByRole('menuitemradio', { name: 'Connected' })
    ).toHaveAttribute('aria-checked', 'true');
  });

  it('opens on the first choice of Data', () => {
    renderMenu({ scenario: 'wrong' });
    open();

    expect(screen.getByRole('menuitemradio', { name: 'Live' })).toHaveFocus();
  });

  it('shows on live data the role and Operations the page shows, not the ignored switch', () => {
    renderMenu(
      { ops: 'on', role: 'admin' },
      {
        role: 'viewer',
        operations: false,
        access: { admin: false, operations: false },
      }
    );

    expect(
      screen.getByRole('button', { name: 'Prototype: Live' })
    ).toBeInTheDocument();

    const menu = open();

    expect(
      within(menu).getByRole('menuitemradio', { name: 'As signed in' })
    ).toHaveAttribute('aria-checked', 'true');
    expect(
      within(menu).getByRole('menuitemradio', { name: /^Admin/ })
    ).toHaveAttribute('aria-disabled', 'true');
    expect(
      within(menu).getByRole('menuitemradio', { name: /^Admin/ })
    ).toHaveTextContent('Sample data only');
    expect(
      within(menu).getByRole('menuitemradio', { name: /^On/ })
    ).toHaveTextContent('Admins only');
  });

  it('lets an admin without Operations turn it on only for sample data', () => {
    renderMenu(
      { ops: 'on' },
      { operations: false, access: { admin: true, operations: false } }
    );

    expect(
      screen.getByRole('button', { name: 'Prototype: Live' })
    ).toBeInTheDocument();
    expect(
      within(open()).getByRole('menuitemradio', { name: /^On/ })
    ).toHaveAttribute('aria-disabled', 'true');
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
