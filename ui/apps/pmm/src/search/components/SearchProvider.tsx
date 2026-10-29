import { type FC, type PropsWithChildren, useEffect } from 'react';
import { useSearchExperiment } from '../search.experiment';
import { resetSearchStore } from '../search.store';
import { SearchRuntimeGate } from './SearchRuntime';

// With search off it adds nothing: no listeners, no queries and no DOM.
export const SearchProvider: FC<PropsWithChildren> = ({ children }) => {
  const enabled = useSearchExperiment();

  useEffect(() => {
    if (!enabled) {
      resetSearchStore();
    }
  }, [enabled]);

  return (
    <>
      {children}
      {enabled && <SearchRuntimeGate />}
    </>
  );
};
