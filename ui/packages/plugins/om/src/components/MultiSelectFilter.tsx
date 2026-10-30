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

import {
  Checkbox,
  ListItemText,
  MenuItem,
  TextField,
  type SelectChangeEvent,
} from '@mui/material';

export interface FilterOption {
  value: string;
  label: string;
}

/**
 * A labelled filter select for a Fleet toolbar, where picking several values widens
 * the match rather than narrowing it. Nothing picked is no filter, and says so.
 */
export const MultiSelectFilter = ({
  label,
  options,
  value,
  onChange,
}: {
  label: string;
  options: FilterOption[];
  value: string[];
  onChange: (value: string[]) => void;
}) => {
  const labelOf = (selected: string) =>
    options.find((option) => option.value === selected)?.label ?? selected;
  return (
    <TextField
      select
      size="small"
      label={label}
      value={value}
      onChange={(event) =>
        onChange(
          (event as SelectChangeEvent<string[]>).target.value as string[]
        )
      }
      slotProps={{
        inputLabel: { shrink: true },
        select: {
          multiple: true,
          displayEmpty: true,
          renderValue: (selected) => {
            const values = selected as string[];
            if (!values.length) {
              return 'Any';
            }
            return values.length === 1
              ? labelOf(values[0])
              : `${values.length} selected`;
          },
        },
      }}
      sx={{ minWidth: 150 }}
    >
      {options.map((option) => (
        <MenuItem key={option.value} value={option.value} dense>
          <Checkbox
            size="small"
            checked={value.includes(option.value)}
            sx={{ p: 0, mr: 1 }}
          />
          <ListItemText primary={option.label} />
        </MenuItem>
      ))}
    </TextField>
  );
};
