import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { getSearchState, takeSearchOpener } from '../search.store';
import {
  mockSearchApis,
  renderWithSearchContexts,
  setPlatform,
} from '../search.testFixtures';
import { enableSearchForTest, hasPeakSearch } from '../testUtils';
import { SidebarSearchEntry } from './SidebarSearchEntry';

const originalPlatform = navigator.platform;

describe('SidebarSearchEntry', () => {
  afterEach(() => {
    setPlatform(originalPlatform);
    vi.restoreAllMocks();
  });

  it('renders nothing and needs no query client while search is off', () => {
    const { container } = render(<SidebarSearchEntry drawerOpen />);

    expect(container).toBeEmptyDOMElement();
  });

  describe.skipIf(!hasPeakSearch)('with Peak UI search, search on', () => {
    enableSearchForTest();

    beforeEach(() => {
      mockSearchApis();
    });

    it.each([
      ['MacIntel', 'Meta+K'],
      ['Win32', 'Control+K'],
    ])('is a Search button with the %s shortcut', (platform, keys) => {
      setPlatform(platform);
      renderWithSearchContexts(<SidebarSearchEntry drawerOpen />);

      const trigger = screen.getByRole('button', { name: 'Search' });

      expect(trigger).toHaveAttribute('aria-haspopup', 'dialog');
      expect(trigger).toHaveAttribute('aria-keyshortcuts', keys);
      expect(trigger).toHaveAttribute('data-testid', 'sidebar-search-trigger');
    });

    it('opens search with itself as the opener', () => {
      renderWithSearchContexts(<SidebarSearchEntry drawerOpen />);
      const trigger = screen.getByRole('button', { name: 'Search' });

      fireEvent.click(trigger);

      expect(getSearchState()).toMatchObject({
        open: true,
        query: '',
        source: 'sidebar',
      });
      expect(takeSearchOpener()).toBe(trigger);
    });

    it('opens search with a character typed on it', () => {
      renderWithSearchContexts(<SidebarSearchEntry drawerOpen />);
      const trigger = screen.getByRole('button', { name: 'Search' });

      fireEvent.keyDown(trigger, { key: 'm' });

      expect(getSearchState()).toMatchObject({ open: true, query: 'm' });
      expect(takeSearchOpener()).toBe(trigger);
    });

    it('keeps the same element when the drawer collapses', () => {
      const { rerender } = renderWithSearchContexts(
        <SidebarSearchEntry drawerOpen />
      );
      const expanded = screen.getByTestId('sidebar-search-trigger');

      rerender(<SidebarSearchEntry drawerOpen={false} />);

      expect(screen.getByTestId('sidebar-search-trigger')).toBe(expanded);
      expect(expanded).toHaveAccessibleName('Search');
    });
  });
});
