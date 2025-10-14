import { fireEvent, render, screen } from '@testing-library/react';
import { wrapWithRouter } from 'utils/testUtils';
import NavItem from './NavItem';
import { NavItemProps } from './NavItem.types';
import { NavItem as NavTreeItem } from 'types/navigation.types';
import { collapseClasses } from '@mui/material/Collapse';
import { MemoryRouterProps } from 'react-router-dom';

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

const renderNavItem = (
  props?: Partial<NavItemProps>,
  routerProps?: Partial<MemoryRouterProps>
) =>
  render(
    wrapWithRouter(
      <NavItem item={TEST_NAV_TREE} drawerOpen={true} level={0} {...props} />,
      routerProps
    )
  );

describe('NavItem', () => {
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
    renderNavItem({ drawerOpen: false });

    const level0Collapse = screen.queryByTestId('navitem-level-0-collapse');

    expect(level0Collapse).toHaveClass(collapseClasses.hidden);
  });

  it('opens root if child is active', () => {
    renderNavItem({}, { initialEntries: ['/0/10'] });

    const level0Collapse = screen.queryByTestId('navitem-level-0-collapse');
    const level1Collapse = screen.queryByTestId('navitem-level-1-collapse');

    expect(level0Collapse).not.toHaveClass(collapseClasses.hidden);
    expect(level1Collapse).toHaveClass(collapseClasses.hidden);
  });

  it('opens root and parent when child is active', () => {
    renderNavItem({}, { initialEntries: ['/0/1/2'] });

    const level0Collapse = screen.queryByTestId('navitem-level-0-collapse');
    const level1Collapse = screen.queryByTestId('navitem-level-1-collapse');

    expect(level0Collapse).not.toHaveClass(collapseClasses.hidden);
    expect(level1Collapse).not.toHaveClass(collapseClasses.hidden);
  });

  it('renders badge if item has badge', () => {
    renderNavItem({
      item: { id: 'with-badge', badge: { label: 'badge-label' } },
    });

    const badge = screen.getByText('badge-label');
    expect(badge).toBeInTheDocument();
  });

  it('shows dot on root if children has a badge and is hidden', async () => {
    renderNavItem(
      {
        item: {
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
        },
      },
      { initialEntries: ['/root'] }
    );

    fireEvent.click(screen.getByTestId('navitem-with-badge-toggle'));

    expect(screen.getByTestId('navitem-dot')).toBeInTheDocument();
  });
});
