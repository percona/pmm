import { isSearchShortcut } from '@pmm/shared';
import { useEffect } from 'react';
import { isOtherModalOpen } from './search.dom';
import { getSearchState, toggleSearch } from './search.store';

export const useSearchShortcut = () => {
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented || !isSearchShortcut(event)) {
        return;
      }

      // Left alone, so the browser and the open form keep the keys.
      if (!getSearchState().open && isOtherModalOpen()) {
        return;
      }

      // Also keeps the browser's own Ctrl/⌘+K from firing.
      event.preventDefault();

      if (!event.repeat) {
        toggleSearch('shortcut');
      }
    };

    window.addEventListener('keydown', handleKeyDown);

    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);
};
