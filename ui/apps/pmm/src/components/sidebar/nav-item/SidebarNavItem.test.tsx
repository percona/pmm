import { fireEvent, render, screen } from '@testing-library/react';
import { wrapWithRouter } from 'utils/testUtils';
import SidebarNavItem from './SidebarNavItem';
import { NavItemProps } from './SidebarNavItem.types';
import { NavItem as NavTreeItem } from 'types/navigation.types';
import { collapseClasses } from '@mui/material/Collapse';
import { MemoryRouterProps } from 'react-router-dom';
import { ThemeContextProvider, pmmThemeOptions } from '@percona/percona-ui';

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
      icon: 'home',
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
});
