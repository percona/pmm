import { useSyncExternalStore } from 'react';
import { isSearchExperimentEnabled } from './search.experiment';

export type SearchOpenSource = 'sidebar' | 'shortcut' | 'grafana';

export interface SearchOpenOptions {
  query?: string;
  opener?: HTMLElement | null;
}

export interface SearchState {
  open: boolean;
  query: string;
  source: SearchOpenSource | null;
}

const INITIAL_STATE: SearchState = { open: false, query: '', source: null };

let state = INITIAL_STATE;
let opener: HTMLElement | null = null;

const listeners = new Set<() => void>();

const notify = () => listeners.forEach((listener) => listener());

const setState = (next: Partial<SearchState>) => {
  state = { ...state, ...next };
  notify();
};

const getActiveElement = () => {
  const active = document.activeElement;

  return active instanceof HTMLElement && active !== document.body
    ? active
    : null;
};

export const getSearchState = () => state;

export const subscribeSearch = (listener: () => void) => {
  listeners.add(listener);

  return () => {
    listeners.delete(listener);
  };
};

export const useSearchState = () =>
  useSyncExternalStore(subscribeSearch, getSearchState, getSearchState);

export const openSearch = (
  source: SearchOpenSource,
  options: SearchOpenOptions = {}
) => {
  if (!isSearchExperimentEnabled() || state.open) {
    return;
  }

  opener = options.opener ?? getActiveElement();
  setState({ open: true, query: options.query ?? '', source });
};

// The query stays until the next open, so the closing fade shows no flash.
export const closeSearch = () => {
  if (state.open) {
    setState({ open: false });
  }
};

export const toggleSearch = (
  source: SearchOpenSource,
  options?: SearchOpenOptions
) => {
  if (state.open) {
    closeSearch();
  } else {
    openSearch(source, options);
  }
};

export const setSearchQuery = (query: string) => {
  if (query !== state.query) {
    setState({ query });
  }
};

export const takeSearchOpener = () => {
  const taken = opener;
  opener = null;

  return taken;
};

export const resetSearchStore = () => {
  opener = null;

  if (state !== INITIAL_STATE) {
    state = INITIAL_STATE;
    notify();
  }
};
