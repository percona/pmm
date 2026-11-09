import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { NO_FILTERS, SEARCH_DEBOUNCE_MS } from './AmazonRds.constants';
import type { GroupFilters } from './AmazonRds.types';
import { readFilters, writeFilters } from './AmazonRds.utils';

// The filters live in the URL, so a reload or a shared link keeps them; the search text reaches
// it once typing stops.
export const useRdsFilters = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const filters = useMemo(() => readFilters(searchParams), [searchParams]);
  const [search, setSearch] = useState(filters.query);

  // Follows the URL when it changes elsewhere, such as the menu link that opens the page unfiltered.
  useEffect(() => {
    setSearch(filters.query);
  }, [filters.query]);

  const update = useCallback(
    (changed: Partial<GroupFilters>) =>
      setSearchParams(
        (prev) => writeFilters({ ...readFilters(prev), ...changed }, prev),
        { replace: true }
      ),
    [setSearchParams]
  );

  useEffect(() => {
    if (search === filters.query) {
      return;
    }

    const timer = setTimeout(
      () => update({ query: search }),
      SEARCH_DEBOUNCE_MS
    );

    return () => clearTimeout(timer);
  }, [search, filters.query, update]);

  const clear = () => {
    setSearch('');
    update(NO_FILTERS);
  };

  return { search, setSearch, filters, update, clear };
};
