import { setSearchExperiment } from './search.experiment';
import {
  closeSearch,
  getSearchState,
  openSearch,
  resetSearchStore,
  setSearchQuery,
  subscribeSearch,
  takeSearchOpener,
  toggleSearch,
} from './search.store';
import { enableSearchForTest } from './testUtils';

describe('search.store', () => {
  enableSearchForTest();

  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('opens with the given opener and query', () => {
    const opener = document.createElement('button');

    openSearch('sidebar', { opener, query: 'm' });

    expect(getSearchState()).toEqual({
      open: true,
      query: 'm',
      source: 'sidebar',
    });
    expect(takeSearchOpener()).toBe(opener);
  });

  it('takes the focused element as the opener when none is given', () => {
    const input = document.createElement('input');
    document.body.append(input);
    input.focus();

    openSearch('shortcut');

    expect(getSearchState().query).toBe('');
    expect(takeSearchOpener()).toBe(input);
  });

  it('records no opener when focus is on the body', () => {
    openSearch('grafana');

    expect(takeSearchOpener()).toBeNull();
  });

  it('ignores an open while already open', () => {
    openSearch('sidebar', { query: 'first' });
    openSearch('shortcut', { query: 'second' });

    expect(getSearchState()).toMatchObject({
      query: 'first',
      source: 'sidebar',
    });
  });

  it('ignores an open while the experiment is off', () => {
    setSearchExperiment(false);

    openSearch('sidebar');

    expect(getSearchState().open).toBe(false);
  });

  it('toggles, and keeps the query on close until the next open', () => {
    toggleSearch('shortcut', { query: 'qan' });
    expect(getSearchState().open).toBe(true);

    toggleSearch('shortcut');
    expect(getSearchState()).toMatchObject({ open: false, query: 'qan' });

    openSearch('sidebar');
    expect(getSearchState().query).toBe('');
  });

  it('clears the opener once taken', () => {
    openSearch('sidebar', { opener: document.createElement('button') });

    expect(takeSearchOpener()).not.toBeNull();
    expect(takeSearchOpener()).toBeNull();
  });

  it('notifies subscribers with a new state object on every change', () => {
    const listener = vi.fn();
    const unsubscribe = subscribeSearch(listener);
    const before = getSearchState();

    openSearch('sidebar');
    setSearchQuery('mysql');
    setSearchQuery('mysql');
    closeSearch();
    closeSearch();

    expect(listener).toHaveBeenCalledTimes(3);
    expect(getSearchState()).not.toBe(before);

    unsubscribe();
    resetSearchStore();

    expect(listener).toHaveBeenCalledTimes(3);
    expect(getSearchState()).toEqual({ open: false, query: '', source: null });
  });
});
