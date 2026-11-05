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

import { useState } from 'react';
import { MenuItem, TextField } from '@mui/material';

/** No grouping: the select's value when the table is flat. */
export const NO_GROUPING = 'none';

export interface GroupByOption {
  /** The column id to group by, or {@link NO_GROUPING}. */
  value: string;
  label: string;
}

const readSession = (key: string) => {
  try {
    return sessionStorage.getItem(key);
  } catch {
    return null;
  }
};

const writeSession = (key: string, value: string) => {
  try {
    sessionStorage.setItem(key, value);
  } catch {
    // Blocked storage only costs the choice outliving the tab switch.
  }
};

/**
 * A Fleet table's grouping, kept for the browser session.
 *
 * Component state alone would not do: switching Fleet tabs unmounts the other tab, and
 * opening a dashboard unmounts the page, while grouping by Health is a triage mode a
 * reader moves in and out of. Not in the URL: a shareable filter matters, a shareable
 * grouping does not (PMM-15767).
 */
export function useSessionGrouping<T extends string>(
  storageKey: string,
  options: GroupByOption[],
  fallback: T
) {
  const [grouping, setGrouping] = useState<T>(() => {
    const stored = readSession(storageKey);
    // A value no option offers, e.g. one renamed since it was stored, is ignored.
    return options.some((option) => option.value === stored)
      ? (stored as T)
      : fallback;
  });
  const update = (value: T) => {
    setGrouping(value);
    writeSession(storageKey, value);
  };
  return [grouping, update] as const;
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
