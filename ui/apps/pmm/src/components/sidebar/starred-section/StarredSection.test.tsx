import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ThemeContextProvider, pmmThemeOptions } from '@percona/peak-ui';
import { NavItem } from 'types/navigation.types';
import { wrapWithRouter } from 'utils/testUtils';
import StarredSection, { STARRED_ORDER_KEY } from './StarredSection';

const HEADING: NavItem = {
  id: 'section-starred',
  type: 'menu-section',
  text: 'Starred',
};

const ITEMS: NavItem[] = [
  { id: 'first', text: 'First', url: '/first', pinned: true },
  { id: 'second', text: 'Second', url: '/second', pinned: true },
  { id: 'third', text: 'Third', url: '/third', pinned: true },
];

const section = (items = ITEMS, drawerOpen = true) => (
  <ThemeContextProvider themeOptions={pmmThemeOptions}>
    {wrapWithRouter(
      <StarredSection
        heading={HEADING}
        items={items}
        activeItem={{ id: 'none' }}
        drawerOpen={drawerOpen}
      />
    )}
  </ThemeContextProvider>
);

const renderSection = (drawerOpen = true) => render(section(ITEMS, drawerOpen));

const savedOrder = () =>
  JSON.parse(localStorage.getItem(STARRED_ORDER_KEY) ?? '[]');

const FOURTH: NavItem = {
  id: 'fourth',
  text: 'Fourth',
  url: '/fourth',
  pinned: true,
};

const entryNames = () =>
  within(screen.getByRole('list', { name: 'Starred' }))
    .getAllByRole('link', { hidden: true })
    .map((link) => link.textContent);

describe('StarredSection', () => {
  beforeEach(() => {
    localStorage.removeItem(STARRED_ORDER_KEY);
  });

  it('keeps its entries as links until editing starts', () => {
    renderSection();

    // Present for keyboard and screen readers, shown on hover or focus.
    expect(
      screen.getByRole('button', { name: 'Edit order' })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /^Reorder/ })
    ).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'First' })).not.toHaveAttribute(
      'aria-disabled'
    );
  });

  it('says how to add an entry while there is none, and leaves the rail without it', async () => {
    const { rerender } = render(section([]));

    expect(screen.getByTestId('starred-empty')).toHaveTextContent(
      'Click the star on apps or dashboards to add them here.'
    );
    expect(
      screen.queryByRole('button', { name: 'Edit order' })
    ).not.toBeInTheDocument();

    rerender(section([], false));

    // It folds away with the drawer, heading included: the rail would show two dividers together.
    await waitFor(() =>
      expect(
        screen.queryByRole('list', { name: 'Starred' })
      ).not.toBeInTheDocument()
    );
    expect(screen.getByTestId('starred-section')).toHaveAttribute(
      'role',
      'none'
    );
  });

  it('offers to edit the order only once there are two entries', () => {
    const { rerender } = render(section(ITEMS.slice(0, 1)));

    expect(screen.queryByTestId('starred-empty')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Edit order' })
    ).not.toBeInTheDocument();

    rerender(section(ITEMS.slice(0, 2)));

    expect(
      screen.getByRole('button', { name: 'Edit order' })
    ).toBeInTheDocument();
  });

  it('turns the entries into rows to reorder, and back with the same button', () => {
    renderSection();

    const toggle = screen.getByRole('button', { name: 'Edit order' });
    fireEvent.click(toggle);

    expect(toggle).toHaveAccessibleName('Done editing order');
    expect(screen.getAllByRole('button', { name: /^Reorder/ })).toHaveLength(3);
    expect(screen.getByTestId('navitem-first')).toHaveAttribute(
      'aria-disabled',
      'true'
    );

    fireEvent.click(toggle);

    expect(toggle).toHaveAccessibleName('Edit order');
    expect(
      screen.queryByRole('button', { name: /^Reorder/ })
    ).not.toBeInTheDocument();
  });

  it('moves an entry with the arrow keys, announces it and keeps the order', () => {
    renderSection();
    fireEvent.click(screen.getByRole('button', { name: 'Edit order' }));

    fireEvent.keyDown(screen.getByRole('button', { name: 'Reorder First' }), {
      key: 'ArrowDown',
    });

    expect(entryNames()).toEqual(['Second', 'First', 'Third']);
    expect(
      screen.getByText('First moved to position 2 of 3.')
    ).toBeInTheDocument();
    expect(JSON.parse(localStorage.getItem(STARRED_ORDER_KEY) ?? '[]')).toEqual(
      ['second', 'first', 'third']
    );
  });

  it('explains a handle on hover, and hides that once it is pressed', async () => {
    renderSection();
    fireEvent.click(screen.getByRole('button', { name: 'Edit order' }));
    const handle = screen.getByRole('button', { name: 'Reorder First' });

    fireEvent.mouseOver(handle);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Drag or click to reorder'
    );

    fireEvent.pointerDown(handle);

    await waitFor(() =>
      expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    );
  });

  it('moves an entry from the handle menu without dragging', () => {
    renderSection();
    fireEvent.click(screen.getByRole('button', { name: 'Edit order' }));

    fireEvent.click(screen.getByRole('button', { name: 'Reorder Third' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Move up' }));

    expect(entryNames()).toEqual(['First', 'Third', 'Second']);
  });

  it('stops editing on Escape', () => {
    renderSection();
    fireEvent.click(screen.getByRole('button', { name: 'Edit order' }));

    fireEvent.keyDown(screen.getByRole('button', { name: 'Reorder Second' }), {
      key: 'Escape',
    });

    // The handle that had focus is gone, so focus returns to the toggle.
    expect(screen.getByRole('button', { name: 'Edit order' })).toHaveFocus();
  });

  it('applies a saved order', () => {
    localStorage.setItem(STARRED_ORDER_KEY, JSON.stringify(['third', 'first']));
    renderSection();

    expect(entryNames()).toEqual(['Third', 'First', 'Second']);
  });

  it('puts a newly starred entry last, wherever it arrives in the list', () => {
    const { rerender } = render(section());

    rerender(section([FOURTH, ...ITEMS]));

    expect(entryNames()).toEqual(['First', 'Second', 'Third', 'Fourth']);
    expect(savedOrder()).toEqual(['first', 'second', 'third', 'fourth']);
  });

  it('forgets an entry that goes away, so it comes back last', () => {
    const { rerender } = render(section());

    rerender(section([ITEMS[0], ITEMS[2]]));
    expect(savedOrder()).toEqual(['first', 'third']);

    rerender(section(ITEMS));
    expect(entryNames()).toEqual(['First', 'Third', 'Second']);
  });

  it('keeps the saved places of entries that have not loaded yet', () => {
    localStorage.setItem(
      STARRED_ORDER_KEY,
      JSON.stringify(['third', 'first', 'second'])
    );
    const { rerender } = render(section([ITEMS[0]]));

    rerender(section(ITEMS));

    expect(entryNames()).toEqual(['Third', 'First', 'Second']);
  });

  it('offers no editing in the collapsed rail', () => {
    renderSection(false);

    expect(
      screen.queryByRole('button', { name: 'Edit order' })
    ).not.toBeInTheDocument();
  });
});
