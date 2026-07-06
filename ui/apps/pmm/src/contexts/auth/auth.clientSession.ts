import { useSyncExternalStore } from 'react';

const CLIENT_SESSION_KEY = 'pmm-ui.session.active';
const SESSION_CHANGE_EVENT = 'pmm-client-session-change';

const PMM_STORAGE_PREFIXES = ['pmm-ui.', 'grafana.'] as const;

const isTrackedStorageKey = (key: string) =>
  key === CLIENT_SESSION_KEY ||
  PMM_STORAGE_PREFIXES.some((prefix) => key.startsWith(prefix));

const notifySessionChange = () => {
  window.dispatchEvent(new Event(SESSION_CHANGE_EVENT));
};

export const establishClientSession = () => {
  if (isClientSessionEstablished()) {
    return;
  }

  localStorage.setItem(CLIENT_SESSION_KEY, 'true');
  notifySessionChange();
};

export const clearClientSession = () => {
  if (!isClientSessionEstablished()) {
    return;
  }

  localStorage.removeItem(CLIENT_SESSION_KEY);
  if (!listenerInstalled) {
    notifySessionChange();
  }
};

export const isClientSessionEstablished = () =>
  localStorage.getItem(CLIENT_SESSION_KEY) === 'true';

export const isGrafanaLoginPath = (pathname: string | null | undefined) =>
  Boolean(pathname?.includes('/login'));

const subscribeToClientSession = (onChange: () => void) => {
  window.addEventListener(SESSION_CHANGE_EVENT, onChange);
  window.addEventListener('storage', onChange);
  return () => {
    window.removeEventListener(SESSION_CHANGE_EVENT, onChange);
    window.removeEventListener('storage', onChange);
  };
};

let listenerInstalled = false;

/**
 * Same-tab localStorage.clear() does not emit "storage".
 * Wrap clear/removeItem so React can react when the user clears site data.
 */
export const ensureClientSessionListener = () => {
  if (typeof window === 'undefined' || listenerInstalled) {
    return;
  }

  listenerInstalled = true;

  const { clear, removeItem } = Storage.prototype;

  Storage.prototype.clear = function patchedClear(this: Storage) {
    clear.call(this);
    if (this === localStorage) {
      notifySessionChange();
    }
  };

  Storage.prototype.removeItem = function patchedRemoveItem(
    this: Storage,
    key: string
  ) {
    removeItem.call(this, key);
    if (this === localStorage && isTrackedStorageKey(key)) {
      notifySessionChange();
    }
  };
};

export const useClientSession = () =>
  useSyncExternalStore(
    subscribeToClientSession,
    isClientSessionEstablished,
    () => false
  );
