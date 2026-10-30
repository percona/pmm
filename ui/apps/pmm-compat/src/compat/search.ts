import {
  isSearchShortcut,
  type CrossFrameMessenger,
  type SearchConfigMessage,
} from '@pmm/shared';
import { triggerShortcut } from 'lib/utils/shortcuts';

export type SearchMessenger = Pick<
  CrossFrameMessenger,
  'addListener' | 'sendMessage'
>;

type Focusable = Element & HTMLOrSVGElement;

interface PendingFocus {
  element: Focusable;
  pathname: string;
}

// Grafana's command palette (kbar) renders this listbox only while it is open.
const KBAR_LISTBOX_ID = 'kbar-listbox';

let messenger: SearchMessenger | undefined;
let enabled = false;
let pending: PendingFocus | null = null;

const isFocusable = (element: Element | null): element is Focusable =>
  element instanceof HTMLElement || element instanceof SVGElement;

const clearPending = () => {
  window.removeEventListener('focus', restoreFocus);
  document.removeEventListener('pointerdown', clearPending, { capture: true });
  pending = null;
};

const restoreFocus = () => {
  const record = pending;

  requestAnimationFrame(() => {
    // Cleared or replaced meanwhile by a pointerdown, another open or search turning off.
    if (!record || record !== pending) {
      return;
    }

    clearPending();

    const active = document.activeElement;
    const focusIsUnset = !active || active === document.body;

    if (
      record.element.isConnected &&
      record.pathname === window.location.pathname &&
      focusIsUnset
    ) {
      record.element.focus({ preventScroll: true });
    }
  });
};

const requestOpen = () => {
  clearPending();

  const active = document.activeElement;

  if (isFocusable(active) && active !== document.body) {
    pending = { element: active, pathname: window.location.pathname };
    window.addEventListener('focus', restoreFocus, { once: true });
    document.addEventListener('pointerdown', clearPending, {
      capture: true,
      once: true,
    });
  }

  messenger?.sendMessage({ type: 'OPEN_SEARCH' });
};

const handleKeyDown = (event: KeyboardEvent) => {
  if (!enabled || event.defaultPrevented || !isSearchShortcut(event)) {
    return;
  }

  // Leave the shortcut to kbar while its palette is open, so it can close itself.
  if (document.getElementById(KBAR_LISTBOX_ID)) {
    return;
  }

  // Stops the event before it reaches kbar's listener on window.
  event.preventDefault();
  event.stopPropagation();

  if (!event.repeat) {
    requestOpen();
  }
};

const setEnabled = (next: boolean) => {
  if (next === enabled) {
    return;
  }

  enabled = next;

  if (enabled) {
    document.addEventListener('keydown', handleKeyDown);
  } else {
    document.removeEventListener('keydown', handleKeyDown);
    clearPending();
  }
};

/** Hands Grafana's search shortcut and toolbar icon to PMM UI while it has search enabled. */
export const initSearch = (searchMessenger: SearchMessenger) => {
  messenger = searchMessenger;
  messenger.addListener({
    type: 'SEARCH_CONFIG',
    onMessage: (message: SearchConfigMessage) =>
      setEnabled(message.payload?.enabled === true),
  });
};

/** Opens PMM search while it is enabled, and Grafana's palette otherwise. */
export const openSearch = () => {
  if (enabled && messenger) {
    requestOpen();
  } else {
    triggerShortcut('search');
  }
};

export const resetSearchForTests = () => {
  setEnabled(false);
  clearPending();
  messenger = undefined;
};
