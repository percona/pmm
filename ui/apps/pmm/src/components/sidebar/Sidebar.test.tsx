import { afterEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { NavigationContext } from 'contexts/navigation/navigation.context';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import type { NavItem } from 'types/navigation.types';
import { Sidebar } from './Sidebar';

const renderSidebar = (
  navOpen: boolean,
  navTree: NavItem[] = [],
  initialEntries = ['/']
) =>
  render(
    <TestWrapper routerProps={{ initialEntries }}>
      {wrapWithQueryProvider(
        <NavigationContext.Provider
          value={{ navTree, navOpen, setNavOpen: () => {} }}
        >
          <Sidebar />
        </NavigationContext.Provider>
      )}
    </TestWrapper>
  );

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
      expect(label.style.getPropertyValue('--sidebar-label-width')).toBe('')
    );

    fireEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }));

    labels.forEach((label) =>
      expect(label.style.getPropertyValue('--sidebar-label-width')).toMatch(
        /^\d+(\.\d+)?px$/
      )
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

    expect(label.style.getPropertyValue('--sidebar-label-width')).toMatch(
      /^\d+(\.\d+)?px$/
    );
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
