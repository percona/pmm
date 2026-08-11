import { useSearchParams } from 'react-router-dom';
import {
  usePerconaTableUrlState,
  type UsePerconaTableUrlStateOptions,
  type UsePerconaTableUrlStateResult,
} from '@percona/percona-ui';

export type UseTableUrlStateOptions = Omit<
  UsePerconaTableUrlStateOptions,
  'searchParams' | 'setSearchParams'
>;

export type UseTableUrlStateResult = UsePerconaTableUrlStateResult;

export const useTableUrlState = (
  options: UseTableUrlStateOptions = {}
): UseTableUrlStateResult => {
  const [searchParams, setSearchParams] = useSearchParams();

  return usePerconaTableUrlState({
    searchParams,
    setSearchParams,
    ...options,
  });
};
