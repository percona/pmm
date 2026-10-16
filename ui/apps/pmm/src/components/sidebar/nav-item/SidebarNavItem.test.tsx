import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { wrapWithRouter } from 'utils/testUtils';
import SidebarNavItem from './SidebarNavItem';
import { NavItemProps } from './SidebarNavItem.types';
import { NavItem as NavTreeItem } from 'types/navigation.types';
import { collapseClasses } from '@mui/material/Collapse';
import { MemoryRouterProps } from 'react-router-dom';
import { ThemeContextProvider, pmmThemeOptions } from '@percona/peak-ui';

const TEST_NAV_TREE: NavTreeItem = {
  id: 'level-0',
  text: 'level-0',
  url: '/0',
  children: [
    {
      id: 'level-10',
      text: 'level-10',
      url: '/0/10',
    },
    {
      id: 'level-1',
      text: 'level-1',
      url: '/0/1',
      children: [
        {
          id: 'level-2',
          text: 'level-2',
          url: '/0/1/2',
        },
      ],
    },
  ],
};

const renderNavItem = ({
  props,
  routerProps,
  activeItem = {
    id: 'not-found',
    url: '/not-found',
  },
}: {
  props?: Partial<NavItemProps>;
  routerProps?: Partial<MemoryRouterProps>;
  activeItem?: NavTreeItem;
} = {}) =>
  render(
    <ThemeContextProvider themeOptions={pmmThemeOptions}>
      {wrapWithRouter(
        <SidebarNavItem
          activeItem={activeItem}
          item={TEST_NAV_TREE}
          drawerOpen={true}
          level={0}
          {...props}
        />,
        routerProps
      )}
    </ThemeContextProvider>
  );

describe('SidebarNavItem', () => {
  it('inner levels are closed by default', () => {
    renderNavItem();

    const level0Collapse = screen.queryByTestId('navitem-level-0-collapse');
    const level1Collapse = screen.queryByTestId('navitem-level-1-collapse');

    expect(level0Collapse).toHaveClass(collapseClasses.hidden);
    expect(level1Collapse).toHaveClass(collapseClasses.hidden);
  });

  it('opens level when item is clicked', async () => {
    renderNavItem();

    const level0 = screen.queryByTestId('navitem-level-0');
    const level0Collapse = screen.queryByTestId('navitem-level-0-collapse');
    const level1Collapse = screen.queryByTestId('navitem-level-1-collapse');

    fireEvent.click(level0!);

    expect(level0Collapse).not.toHaveClass(collapseClasses.hidden);
    expect(level1Collapse).toHaveClass(collapseClasses.hidden);
  });

  it('opens inner level when previous one are clicked', async () => {
    renderNavItem();

    const level0 = screen.queryByTestId('navitem-level-0');
    const level1 = screen.queryByTestId('navitem-level-1');

    const level0Collapse = screen.queryByTestId('navitem-level-0-collapse');
    const level1Collapse = screen.queryByTestId('navitem-level-1-collapse');

    fireEvent.click(level0!);

    fireEvent.click(level1!);

    expect(level0Collapse).not.toHaveClass(collapseClasses.hidden);
    expect(level1Collapse).not.toHaveClass(collapseClasses.hidden);
  });

  it('collapses root items when sidebar is closed', () => {
    renderNavItem({
      props: {
        drawerOpen: false,
      },
    });

    const level0Collapse = screen.queryByTestId('navitem-level-0-collapse');

    expect(level0Collapse).toHaveClass(collapseClasses.hidden);
  });

  it('opens root if child is active', () => {
    renderNavItem({
      activeItem: TEST_NAV_TREE.children![0],
    });

    const level0Collapse = screen.queryByTestId('navitem-level-0-collapse');
    const level1Collapse = screen.queryByTestId('navitem-level-1-collapse');

    expect(level0Collapse).not.toHaveClass(collapseClasses.hidden);
    expect(level1Collapse).toHaveClass(collapseClasses.hidden);
  });

  it('opens root and parent when child is active', () => {
    renderNavItem({
      activeItem: TEST_NAV_TREE.children![1].children![0],
    });

    const level0Collapse = screen.queryByTestId('navitem-level-0-collapse');
    const level1Collapse = screen.queryByTestId('navitem-level-1-collapse');

    expect(level0Collapse).not.toHaveClass(collapseClasses.hidden);
    expect(level1Collapse).not.toHaveClass(collapseClasses.hidden);
  });

  it('renders badge if item has badge', () => {
    renderNavItem({
      props: {
        item: { id: 'with-badge', badge: { label: 'badge-label' } },
      },
    });

    const badge = screen.getByText('badge-label');
    expect(badge).toBeInTheDocument();
  });

  it('shows dot on root if children has a badge and is hidden', async () => {
    const item: NavTreeItem = {
      id: 'with-badge',
      icon: 'pmm-rounded',
      url: '/root',
      children: [
        {
          id: 'with-badge-child',
          url: '/root/child',
          badge: { label: 'badge-label' },
        },
      ],
    };
    const { container } = renderNavItem({
      activeItem: item,
      props: { item },
    });

    fireEvent.click(screen.getByTestId('navitem-with-badge-toggle'));

    expect(container.querySelector('.MuiBadge-dot')).toBeInTheDocument();
  });

  it('renders divider if item has type "menu-divider"', () => {
    renderNavItem({
      props: {
        item: { id: 'divider', type: 'menu-divider' },
      },
    });

    expect(screen.queryByTestId('navitem-divider-divider')).toBeInTheDocument();
  });

  it('renders text item if item has type "menu-text"', () => {
    renderNavItem({
      props: {
        item: {
          id: 'desc',
          type: 'menu-text',
          text: 'description',
          secondaryText: 'secondary text',
        },
      },
    });

    expect(screen.queryByTestId('navitem-desc-text-item')).toBeInTheDocument();
    expect(screen.getByText('description')).toBeInTheDocument();
    expect(screen.getByText('secondary text')).toBeInTheDocument();
  });

  it('links a group with a page of its own to that page', () => {
    renderNavItem();

    expect(screen.getByTestId('navitem-level-0')).toHaveAttribute('href', '/0');
  });

  it('links a group without a page to its first child', () => {
    renderNavItem({ props: { item: { ...TEST_NAV_TREE, url: undefined } } });

    expect(screen.getByTestId('navitem-level-0')).toHaveAttribute(
      'href',
      '/0/10'
    );
  });

  it('renders a quick action beside a leaf, outside its link', () => {
    renderNavItem({
      props: {
        item: {
          id: 'services',
          text: 'Services',
          url: '/services',
          action: { label: 'Add service', url: '/add' },
        },
      },
    });

    const action = screen.getByTestId('navitem-services-action');

    expect(action).toHaveAttribute('href', '/add');
    expect(action).toHaveAccessibleName('Add service');
    expect(screen.getByTestId('navitem-services')).not.toContainElement(action);
  });

  it('hides the quick action when the sidebar is collapsed', () => {
    renderNavItem({
      props: {
        drawerOpen: false,
        item: {
          id: 'services',
          text: 'Services',
          url: '/services',
          action: { label: 'Add service', url: '/add' },
        },
      },
    });

    expect(
      screen.queryByTestId('navitem-services-action')
    ).not.toBeInTheDocument();
  });

  it('shows no reorder handle on a pinned entry outside editing', () => {
    renderNavItem({
      props: { item: { id: 'pin', text: 'Pinned', url: '/pin', pinned: true } },
    });

    expect(screen.queryByTestId('navitem-pin-handle')).not.toBeInTheDocument();
  });

  it('places a trailing control outside the entry and stops its link while editing', () => {
    renderNavItem({
      props: {
        item: { id: 'pin', text: 'Pinned', url: '/pin', pinned: true },
        trailingControl: <button type="button">Reorder Pinned</button>,
        disableLink: true,
      },
    });

    const link = screen.getByTestId('navitem-pin');
    expect(link).not.toContainElement(
      screen.getByRole('button', { name: 'Reorder Pinned' })
    );
    expect(link).toHaveAttribute('aria-disabled', 'true');
    expect(link).toHaveAttribute('tabindex', '-1');
  });

  it('renders a section action beside the heading', () => {
    renderNavItem({
      props: {
        item: {
          id: 'mine',
          type: 'menu-section',
          text: 'Starred',
          action: { label: 'Customize navigation', url: '/customize' },
        },
      },
    });

    expect(screen.getByTestId('navitem-mine-action')).toHaveAttribute(
      'href',
      '/customize'
    );
  });

  it('renders a heading if item has type "menu-section"', () => {
    renderNavItem({
      props: {
        item: { id: 'browse', type: 'menu-section', text: 'Browse' },
      },
    });

    expect(screen.getByTestId('navitem-browse-section')).toBeInTheDocument();
    expect(screen.getByText('Browse')).toBeInTheDocument();
  });

  it('renders a heading inside an open group', () => {
    renderNavItem({
      props: {
        item: {
          id: 'group',
          text: 'group',
          url: '/group',
          children: [
            { id: 'first', text: 'first', url: '/group/first' },
            { id: 'heading', type: 'menu-section', text: 'More' },
            { id: 'second', text: 'second', url: '/group/second' },
          ],
        },
      },
    });

    fireEvent.click(screen.getByTestId('navitem-group'));

    const heading = screen.getByTestId('navitem-heading-section');
    expect(heading).toHaveTextContent('More');
    expect(heading.querySelector('a')).toBeNull();
  });

  it('renders a section as a divider when the sidebar is collapsed', () => {
    renderNavItem({
      props: {
        item: { id: 'browse', type: 'menu-section', text: 'Browse' },
        drawerOpen: false,
      },
    });

    expect(screen.getByTestId('navitem-browse-divider')).toBeInTheDocument();
    expect(
      screen.getByText('Browse').closest('[aria-hidden="true"]')
    ).not.toBeNull();
  });

  it('names the group toggle and reports whether the group is open', () => {
    renderNavItem();

    const toggle = screen.getByRole('button', { name: 'level-0 submenu' });
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    expect(toggle).toHaveAttribute('aria-controls', 'navitem-level-0-children');

    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
  });

  it('leaves the selected state to an active child in view', () => {
    const child = TEST_NAV_TREE.children![0];
    renderNavItem({ activeItem: child });

    const group = screen.getByTestId('navitem-level-0');
    expect(screen.getByTestId('navitem-level-10')).toHaveClass('Mui-selected');
    expect(group).not.toHaveClass('Mui-selected');
    expect(group).not.toHaveAttribute('aria-current');
  });

  it('selects the group as the current section when its active child is hidden', () => {
    const child = TEST_NAV_TREE.children![0];
    renderNavItem({ activeItem: child, props: { drawerOpen: false } });

    const group = screen.getByTestId('navitem-level-0');
    expect(group).toHaveClass('Mui-selected');
    expect(group).toHaveAttribute('aria-current', 'true');
  });

  describe('in the collapsed rail', () => {
    it('opens a group in a panel beside it instead of following its link', () => {
      renderNavItem({ props: { drawerOpen: false } });
      const trigger = screen.getByTestId('navitem-level-0');

      expect(trigger).toHaveAttribute('role', 'button');
      expect(trigger).toHaveAttribute('aria-expanded', 'false');

      fireEvent.click(trigger);

      expect(trigger).toHaveAttribute('aria-expanded', 'true');
      const panel = screen.getByRole('navigation', { name: 'level-0' });
      expect(trigger).toHaveAttribute(
        'aria-controls',
        panel.closest('.MuiPopover-root')?.id
      );
      expect(
        within(panel).getByRole('link', { name: 'level-10' })
      ).toBeInTheDocument();
    });

    it('marks the current page in the panel', () => {
      const child = TEST_NAV_TREE.children![0];
      renderNavItem({ props: { drawerOpen: false }, activeItem: child });

      fireEvent.click(screen.getByTestId('navitem-level-0'));

      const panel = screen.getByRole('navigation', { name: 'level-0' });
      expect(within(panel).getByRole('link', { name: 'level-10' })).toHaveClass(
        'Mui-selected'
      );
    });

    it('opens on Space and closes on Escape', async () => {
      renderNavItem({ props: { drawerOpen: false } });
      const trigger = screen.getByTestId('navitem-level-0');

      fireEvent.keyDown(trigger, { key: ' ' });
      const panel = screen.getByRole('navigation', { name: 'level-0' });

      fireEvent.keyDown(panel, { key: 'Escape' });

      await waitFor(() =>
        expect(
          screen.queryByRole('navigation', { name: 'level-0' })
        ).not.toBeInTheDocument()
      );
      expect(trigger).toHaveAttribute('aria-expanded', 'false');
    });

    it('moves between entries with the arrow keys, Home and End', () => {
      renderNavItem({ props: { drawerOpen: false } });
      fireEvent.click(screen.getByTestId('navitem-level-0'));
      const panel = screen.getByRole('navigation', { name: 'level-0' });
      const first = within(panel).getByRole('link', { name: 'level-10' });
      const group = within(panel).getByRole('link', { name: 'level-1' });

      first.focus();
      fireEvent.keyDown(first, { key: 'ArrowDown' });
      expect(group).toHaveFocus();

      // What the closed group holds is skipped, so Down wraps round.
      fireEvent.keyDown(group, { key: 'ArrowDown' });
      expect(first).toHaveFocus();

      fireEvent.keyDown(first, { key: 'ArrowUp' });
      expect(group).toHaveFocus();

      fireEvent.keyDown(group, { key: 'Home' });
      expect(first).toHaveFocus();

      fireEvent.keyDown(first, { key: 'End' });
      expect(group).toHaveFocus();
    });

    it('moves on from the row of a focused action', () => {
      renderNavItem({ props: { drawerOpen: false } });
      fireEvent.click(screen.getByTestId('navitem-level-0'));
      const panel = screen.getByRole('navigation', { name: 'level-0' });
      const toggle = within(panel).getByRole('button', {
        name: 'level-1 submenu',
      });

      toggle.focus();
      fireEvent.keyDown(toggle, { key: 'ArrowUp' });

      expect(
        within(panel).getByRole('link', { name: 'level-10' })
      ).toHaveFocus();
    });

    it('keeps groups opening in place in the expanded sidebar', () => {
      renderNavItem();
      const group = screen.getByTestId('navitem-level-0');

      expect(group).not.toHaveAttribute('role');

      fireEvent.click(group);

      expect(
        screen.queryByRole('navigation', { name: 'level-0' })
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('navitem-level-0-collapse')).not.toHaveClass(
        collapseClasses.hidden
      );
    });
  });
});
