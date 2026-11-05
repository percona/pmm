import { useEffect, useMemo, useState } from 'react';
import { isEqual } from 'lodash';
import { locationService } from '@grafana/runtime';
import { SEARCH_DEBOUNCE_MS } from './AuroraTreePanel.constants';
import { readFilters, toUrlQuery } from './AuroraTreePanel.tools';
import { ClusterFilters } from './AuroraTreePanel.types';

// Filters start from the URL and are written back to it, so a reload or a shared link keeps them.
export const useClusterFilters = () => {
  const [initial] = useState(() => readFilters(locationService.getSearchObject()));
  const [search, setSearch] = useState(initial.query);
  const [query, setQuery] = useState(initial.query);
  const [accounts, setAccounts] = useState(initial.accounts);
  const [clusters, setClusters] = useState(initial.clusters);

  useEffect(() => {
    const timer = setTimeout(() => setQuery(search), SEARCH_DEBOUNCE_MS);

    return () => clearTimeout(timer);
  }, [search]);

  const filters = useMemo((): ClusterFilters => ({ query, accounts, clusters }), [query, accounts, clusters]);

  useEffect(() => {
    if (!isEqual(filters, readFilters(locationService.getSearchObject()))) {
      locationService.partial(toUrlQuery(filters), true);
    }
  }, [filters]);

  const clear = () => {
    setSearch('');
    setQuery('');
    setAccounts([]);
    setClusters([]);
  };

  return {
    search,
    setSearch,
    filters,
    setAccounts,
    setClusters,
    clear,
  };
};
