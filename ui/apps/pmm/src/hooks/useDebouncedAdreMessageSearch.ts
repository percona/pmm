import { useCallback, useEffect, useState } from 'react';

const DEFAULT_DEBOUNCE_MS = 350;

/**
 * Debounced search query for ADRE sidebar: flushes immediately when cleared,
 * and exposes `searchPending` while waiting for debounce or in-flight request.
 */
export function useDebouncedAdreMessageSearch(
  searchLoading: boolean,
  onSearch: (q: string) => void | Promise<void>,
  debounceMs: number = DEFAULT_DEBOUNCE_MS
) {
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');

  useEffect(() => {
    const t = query.trim();
    if (!t) {
      setDebouncedQuery('');
      return;
    }
    const id = window.setTimeout(() => setDebouncedQuery(query), debounceMs);
    return () => window.clearTimeout(id);
  }, [query, debounceMs]);

  useEffect(() => {
    void onSearch(debouncedQuery);
  }, [debouncedQuery, onSearch]);

  const q = query.trim();
  const dq = debouncedQuery.trim();
  const searchPending = q !== '' && (q !== dq || searchLoading);

  const clearQuery = useCallback(() => setQuery(''), []);

  return { query, setQuery, clearQuery, searchPending };
}
