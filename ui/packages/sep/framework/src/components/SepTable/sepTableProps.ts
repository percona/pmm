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

import type { Theme } from '@mui/material/styles';
import type { MaterialReactTableProps } from 'material-react-table';
import { SEP_TABLE_CLASS } from '../../constants';

/**
 * Opaque table surface. The Percona theme's `background.paper` doesn't always
 * resolve to an opaque colour, leaving the table looking transparent against
 * tinted page backgrounds, so pick the mode's own opaque surface instead of
 * pinning `common.white` (which renders a white table in dark mode).
 */
export const opaqueTableSurface = (theme: Theme) => ({
  bgcolor:
    theme.palette.mode === 'dark'
      ? theme.palette.background.default
      : theme.palette.common.white,
});

/**
 * The presentation every list in a SEP app shares.
 *
 * There are three lists — the plugin list, a task's run history, and a task's
 * schedules — and they used to be a data table, a data table with different
 * chrome, and a hand-rolled `<Table>` inside a card. A reader who learns one
 * list learned only that one (PMM-15456). This is the single answer to what a
 * SEP list looks like; a caller adds its columns, its data, and its own header
 * actions, and overrides nothing else without a reason.
 *
 * Column visibility is left to the caller: a list whose columns come from a
 * schema has something to hide, and one with six fixed columns does not.
 *
 * Deliberately not exported as a component: the three differ in what they fetch
 * and in row behaviour, and a wrapper thin enough to accommodate that would
 * pass every one of these through anyway.
 */
export function sepTableProps<T extends Record<string, unknown>>(): Omit<
  MaterialReactTableProps<T>,
  'columns' | 'data'
> {
  return {
    enableColumnActions: false,
    enableDensityToggle: false,
    enableFullScreenToggle: false,
    // A column neither stretches to its widest value nor forces the table past
    // its container — the two ways a semantic table ends up scrolling sideways.
    layoutMode: 'grid',
    muiTablePaperProps: {
      className: SEP_TABLE_CLASS,
      elevation: 0,
      variant: 'outlined',
      sx: opaqueTableSurface,
    },
    muiTableContainerProps: {
      sx: opaqueTableSurface,
    },
    muiTableHeadCellProps: {
      sx: {
        // The label, not the sort control: a long header would otherwise set
        // the column's width for every row under it.
        '& .Mui-TableHeadCell-Content-Wrapper': {
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        },
      },
    },
    muiTableBodyCellProps: {
      sx: {
        // Grid layout already hides cell overflow, but `textOverflow` is inert
        // without it — stated here so the rule cannot be read as doing
        // something it does not.
        overflow: 'hidden',
        whiteSpace: 'nowrap',
        textOverflow: 'ellipsis',
      },
    },
  };
}
