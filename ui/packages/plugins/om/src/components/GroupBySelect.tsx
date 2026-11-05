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

import { MenuItem, TextField } from '@mui/material';

/** No grouping: the select's value when the table is flat. */
export const NO_GROUPING = 'none';

export interface GroupByOption {
  /** The column id to group by, or {@link NO_GROUPING}. */
  value: string;
  label: string;
}

/**
 * A labelled "Group by" select for a Fleet table's toolbar.
 *
 * The visible replacement for MRT's drag-a-header-to-group, which nothing on screen
 * advertises (PMM-15712).
 */
export const GroupBySelect = ({
  options,
  value,
  onChange,
}: {
  options: GroupByOption[];
  value: string;
  onChange: (value: string) => void;
}) => (
  <TextField
    select
    size="small"
    label="Group by"
    value={value}
    onChange={(event) => onChange(event.target.value)}
    sx={{ minWidth: 160 }}
  >
    {options.map((option) => (
      <MenuItem key={option.value} value={option.value}>
        {option.label}
      </MenuItem>
    ))}
  </TextField>
);
