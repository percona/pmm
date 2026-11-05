import { useEffect, useMemo, useState } from 'react';
import { isEqual } from 'lodash';
import { locationService } from '@grafana/runtime';
import { SEARCH_DEBOUNCE_MS } from './AuroraTreePanel.constants';
import { readFilters, toUrlQuery } from './AuroraTreePanel.tools';
import { ClusterFilters, ClusterSelection } from './AuroraTreePanel.types';

const NO_SELECTION: ClusterSelection = { regions: [], accounts: [], clusters: [] };

// Filters start from the URL and are written back to it, so a reload or a shared link keeps them.
export const useClusterFilters = () => {
  const [{ query: initialQuery, ...initialSelection }] = useState(() => readFilters(
    locationService.getSearchObject(),
  ));
  const [search, setSearch] = useState(initialQuery);
  const [query, setQuery] = useState(initialQuery);
  const [selection, setSelection] = useState<ClusterSelection>(initialSelection);

  useEffect(() => {
    const timer = setTimeout(() => setQuery(search), SEARCH_DEBOUNCE_MS);

    return () => clearTimeout(timer);
  }, [search]);

  const filters = useMemo((): ClusterFilters => ({ query, ...selection }), [query, selection]);

  useEffect(() => {
    if (!isEqual(filters, readFilters(locationService.getSearchObject()))) {
      locationService.partial(toUrlQuery(filters), true);
    }
  }, [filters]);

  const clear = () => {
    setSearch('');
    setQuery('');
    setSelection(NO_SELECTION);
  };

  return {
    search,
    setSearch,
    filters,
    setSelection,
    clear,
  };
};
