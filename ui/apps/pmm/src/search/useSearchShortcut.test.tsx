import { fireEvent, renderHook } from '@testing-library/react';
import { getSearchState, openSearch } from './search.store';
import { enableSearchForTest } from './testUtils';
import { useSearchShortcut } from './useSearchShortcut';

const originalPlatform = navigator.platform;

const setPlatform = (value: string) =>
  Object.defineProperty(navigator, 'platform', { value, configurable: true });

const press = (init: KeyboardEventInit & { keyCode?: number }) => {
  const event = new KeyboardEvent('keydown', {
    bubbles: true,
    cancelable: true,
    ...init,
  });
  fireEvent(window, event);

  return event;
};

describe('useSearchShortcut', () => {
  enableSearchForTest();

  afterEach(() => {
    setPlatform(originalPlatform);
    document.body.innerHTML = '';
  });

  describe('on a Mac', () => {
    beforeEach(() => setPlatform('MacIntel'));

    it('opens with ⌘K and blocks the browser default', () => {
      renderHook(() => useSearchShortcut());

      const event = press({ key: 'k', metaKey: true });

      expect(event.defaultPrevented).toBe(true);
      expect(getSearchState()).toMatchObject({
        open: true,
        source: 'shortcut',
      });
    });

    it('ignores Ctrl+K', () => {
      renderHook(() => useSearchShortcut());

      const event = press({ key: 'k', ctrlKey: true });

      expect(event.defaultPrevented).toBe(false);
      expect(getSearchState().open).toBe(false);
    });

    it('closes when pressed while open', () => {
      renderHook(() => useSearchShortcut());

      press({ key: 'k', metaKey: true });
      press({ key: 'k', metaKey: true });

      expect(getSearchState().open).toBe(false);
    });
  });

  describe('on Windows', () => {
    beforeEach(() => setPlatform('Win32'));

    it('opens with Ctrl+K', () => {
      renderHook(() => useSearchShortcut());

      press({ key: 'k', ctrlKey: true });

      expect(getSearchState().open).toBe(true);
    });

    it.each([
      ['Shift', { shiftKey: true }],
      ['Alt', { altKey: true }],
    ])('ignores the %s variant', (_, modifier) => {
      renderHook(() => useSearchShortcut());

      const event = press({ key: 'k', ctrlKey: true, ...modifier });

      expect(event.defaultPrevented).toBe(false);
      expect(getSearchState().open).toBe(false);
    });

    it('survives a keydown without a key, as Chrome autofill sends', () => {
      renderHook(() => useSearchShortcut());

      expect(() => press({ ctrlKey: true })).not.toThrow();
      expect(getSearchState().open).toBe(false);
    });

    it('swallows a held key without toggling', () => {
      renderHook(() => useSearchShortcut());

      const event = press({ key: 'k', ctrlKey: true, repeat: true });

      expect(event.defaultPrevented).toBe(true);
      expect(getSearchState().open).toBe(false);
    });

    it('ignores an event something else already handled', () => {
      renderHook(() => useSearchShortcut());
      const input = document.createElement('input');
      document.body.append(input);
      input.addEventListener('keydown', (event) => event.preventDefault());

      fireEvent.keyDown(input, { key: 'k', ctrlKey: true });

      expect(getSearchState().open).toBe(false);
    });

    it('leaves the keys alone while another modal dialog is open', () => {
      renderHook(() => useSearchShortcut());
      const dialog = document.createElement('div');
      dialog.setAttribute('role', 'dialog');
      dialog.setAttribute('aria-modal', 'true');
      document.body.append(dialog);

      const event = press({ key: 'k', ctrlKey: true });

      expect(event.defaultPrevented).toBe(false);
      expect(getSearchState().open).toBe(false);
    });

    it('still closes search when its own dialog is the open one', () => {
      renderHook(() => useSearchShortcut());
      openSearch('sidebar');
      const root = document.createElement('div');
      root.dataset.testid = 'pmm-search-dialog';
      root.innerHTML = '<div role="dialog" aria-modal="true"></div>';
      document.body.append(root);

      press({ key: 'k', ctrlKey: true });

      expect(getSearchState().open).toBe(false);
    });

    it('reopens while its own dialog is still fading out', () => {
      renderHook(() => useSearchShortcut());
      const root = document.createElement('div');
      root.dataset.testid = 'pmm-search-dialog';
      root.innerHTML = '<div role="dialog" aria-modal="true"></div>';
      document.body.append(root);

      const event = press({ key: 'k', ctrlKey: true });

      expect(event.defaultPrevented).toBe(true);
      expect(getSearchState()).toMatchObject({
        open: true,
        source: 'shortcut',
      });
    });

    it('stops listening on unmount', () => {
      const { unmount } = renderHook(() => useSearchShortcut());

      unmount();
      const event = press({ key: 'k', ctrlKey: true });

      expect(event.defaultPrevented).toBe(false);
      expect(getSearchState().open).toBe(false);
    });
  });
});
