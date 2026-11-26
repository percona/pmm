/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { useSearchParams } from 'react-router-dom';
import {
  usePerconaTableUrlState,
  type UsePerconaTableUrlStateOptions,
} from '@percona/peak-ui';
import type { MRT_Updater } from 'material-react-table';

export type TableUrlStateOptions = Omit<
  UsePerconaTableUrlStateOptions,
  'searchParams' | 'setSearchParams'
>;

/** A table's search, filters and sort in the URL, as the PMM app's own tables keep them. */
export function useTableUrlState(options: TableUrlStateOptions) {
  const [searchParams, setSearchParams] = useSearchParams();
  const { tableProps } = usePerconaTableUrlState({
    searchParams,
    setSearchParams,
    ...options,
  });
  return {
    ...tableProps,
    // MRT's clear button sets no search at all, which peak-ui writes as "undefined".
    onGlobalFilterChange: (updater: MRT_Updater<string>) =>
      tableProps.onGlobalFilterChange(
        (previous) =>
          (typeof updater === 'function' ? updater(previous) : updater) ?? ''
      ),
  };
}
