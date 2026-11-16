import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  fireEvent,
  render,
  screen,
  waitForElementToBeRemoved,
} from '@testing-library/react';
import { NavigationContext } from 'contexts/navigation/navigation.context';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import type { NavItem } from 'types/navigation.types';
import { useState } from 'react';
import { Sidebar } from './Sidebar';

const StatefulNavigation = ({
  initialOpen,
  navTree,
}: {
  initialOpen: boolean;
  navTree: NavItem[];
}) => {
  const [navOpen, setNavOpen] = useState(initialOpen);

  return (
    <NavigationContext.Provider value={{ navTree, navOpen, setNavOpen }}>
      <Sidebar />
    </NavigationContext.Provider>
  );
};

const renderSidebar = (
  navOpen: boolean,
  navTree: NavItem[] = [],
  initialEntries = ['/']
) =>
  render(
    <TestWrapper routerProps={{ initialEntries }}>
      {wrapWithQueryProvider(
        <StatefulNavigation initialOpen={navOpen} navTree={navTree} />
      )}
    </TestWrapper>
  );

const labelWidth = (label: Element) =>
  getComputedStyle(label).getPropertyValue('--sidebar-label-width').trim();

describe('Sidebar', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("records each label's expanded width as the drawer starts to close", () => {
    renderSidebar(true, [
      { id: 'overview', text: 'Overview', url: '/graph/' },
      { id: 'summary', text: 'Databases on Kubernetes - Summary', url: '/x' },
    ]);
    const labels = screen
      .getByTestId('pmm-sidebar')
      .querySelectorAll<HTMLElement>('.MuiListItemText-root');

    expect(labels).toHaveLength(2);
    labels.forEach((label) =>
      expect(labelWidth(label)).not.toMatch(/^\d+(\.\d+)?px$/)
    );

    fireEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }));

    labels.forEach((label) =>
      expect(labelWidth(label)).toMatch(/^\d+(\.\d+)?px$/)
    );
  });

  it('records the widths too when a narrow layout closes after a click', () => {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query.includes('max-width'),
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    }));
    renderSidebar(true, [{ id: 'overview', text: 'Overview', url: '/x' }]);
    const [label] = screen
      .getByTestId('pmm-sidebar')
      .querySelectorAll<HTMLElement>('.MuiListItemText-root');

    fireEvent.click(screen.getByRole('link', { name: 'Overview' }));

    expect(labelWidth(label)).toMatch(/^\d+(\.\d+)?px$/);
  });

  it('keeps a recorded width for an entry that remounts while collapsed', async () => {
    renderSidebar(
      true,
      [
        {
          id: 'mysql',
          text: 'MySQL',
          url: '/graph/d/mysql',
          children: [
            { id: 'mysql-summary', text: 'Summary', url: '/graph/d/summary' },
          ],
        },
      ],
      ['/graph/d/summary']
    );

    fireEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }));

    await waitForElementToBeRemoved(() =>
      screen.queryByTestId('navitem-mysql-summary')
    );

    fireEvent.click(screen.getByRole('button', { name: 'Expand sidebar' }));

    const label = screen
      .getByTestId('navitem-mysql-summary')
      .querySelector('.MuiListItemText-root');
    expect(labelWidth(label as Element)).toMatch(/^\d+(\.\d+)?px$/);
  });

  it('clears the highlight on the home dashboard, which has no entry', () => {
    renderSidebar(
      true,
      [{ id: 'mysql', text: 'MySQL', url: '/graph/d/mysql' }],
      ['/graph/d/mysql']
    );
    const entry = screen.getByRole('link', { name: 'MySQL' });

    expect(entry).toHaveClass('Mui-selected');

    fireEvent.click(screen.getByRole('link', { name: 'Home' }));

    expect(entry).not.toHaveClass('Mui-selected');
  });
});
