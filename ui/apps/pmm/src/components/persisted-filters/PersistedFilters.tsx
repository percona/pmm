import { type FC, type PropsWithChildren, useEffect } from 'react';
import {
  Navigate,
  useLocation,
  useNavigationType,
  useSearchParams,
} from 'react-router-dom';

interface Props extends PropsWithChildren {
  storageKey: string;
  params: string[];
}

// Saves the page's filter and page-size params and restores them when the page
// is opened without a query string (e.g. from the sidebar); other URLs are left as is.
const PersistedFilters: FC<Props> = ({ storageKey, params, children }) => {
  const { search } = useLocation();
  const navigationType = useNavigationType();
  const [searchParams] = useSearchParams();
  // pages update their params with replace, so clearing them never restores
  const restore =
    !search && navigationType !== 'REPLACE'
      ? (localStorage.getItem(storageKey) ?? '')
      : '';

  useEffect(() => {
    if (restore) {
      return;
    }
    const filters = new URLSearchParams();
    params.forEach((param) => {
      const value = searchParams.get(param);
      if (value) {
        filters.set(param, value);
      }
    });
    localStorage.setItem(storageKey, filters.toString());
  }, [restore, searchParams, storageKey, params]);

  // children wait for the restored URL so the page never fetches unfiltered
  return restore ? <Navigate to={{ search: restore }} replace /> : children;
};

export default PersistedFilters;
