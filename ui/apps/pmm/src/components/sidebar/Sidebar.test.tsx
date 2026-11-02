import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { NavigationContext } from 'contexts/navigation/navigation.context';
import { enableSearchForTest, hasPeakSearch } from 'search/testUtils';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import type { NavItem } from 'types/navigation.types';
import { Sidebar } from './Sidebar';

const renderSidebar = (navOpen: boolean, navTree: NavItem[] = []) =>
  render(
    <TestWrapper>
      {wrapWithQueryProvider(
        <NavigationContext.Provider
          value={{ navTree, navOpen, setNavOpen: () => {} }}
        >
          <Sidebar />
        </NavigationContext.Provider>
      )}
    </TestWrapper>
  );

// The drawer paper's own children, from the heading down to the list's container.
const getPaperChildren = () => {
  const paper = screen
    .getByTestId('pmm-sidebar')
    .querySelector<HTMLElement>('.MuiDrawer-paper');

  return Array.from(paper?.children ?? []);
};

describe('Sidebar', () => {
  it.each([true, false])(
    'has no search entry while search is off (drawer open: %s)',
    (navOpen) => {
      renderSidebar(navOpen);
      const children = getPaperChildren();

      expect(
        screen.queryByTestId('sidebar-search-trigger')
      ).not.toBeInTheDocument();
      expect(children).toHaveLength(2);
      expect(children[1]).toContainElement(screen.getByRole('list'));
    }
  );

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

  describe.skipIf(!hasPeakSearch)('with Peak UI search, search on', () => {
    enableSearchForTest();

    it('makes search the first entry of the list', () => {
      renderSidebar(true);
      const children = getPaperChildren();
      const entries = screen.getAllByRole('listitem');

      expect(children).toHaveLength(2);
      expect(entries[0]).toContainElement(
        screen.getByTestId('sidebar-search-trigger')
      );
    });
  });
});
