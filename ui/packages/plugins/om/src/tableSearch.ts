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

import type { MRT_RowData, MRT_TableOptions } from 'material-react-table';

/**
 * Table options that search the columns setting `enableGlobalFilter: true` by plain
 * substring, rather than MRT's fuzzy match over every column, hidden ones included.
 */
export function substringSearch<T extends MRT_RowData>(
  label: string
): Partial<MRT_TableOptions<T>> {
  return {
    globalFilterFn: 'contains',
    defaultColumn: { enableGlobalFilter: false },
    // TanStack only searches a column whose first row holds a string.
    getColumnCanGlobalFilter: () => true,
    enableFilterMatchHighlighting: false,
    muiSearchTextFieldProps: {
      placeholder: label,
      inputProps: { 'aria-label': label },
    },
  };
}
