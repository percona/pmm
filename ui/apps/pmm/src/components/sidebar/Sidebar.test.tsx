import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { NavigationContext } from 'contexts/navigation/navigation.context';
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

describe('Sidebar', () => {
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
});
