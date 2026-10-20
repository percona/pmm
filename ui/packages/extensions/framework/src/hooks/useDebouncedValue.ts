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

import { useEffect, useState } from 'react';

/**
 * Default settle window for a typed input before it drives work downstream (ms).
 *
 * Named for its origin — the search boxes across the apps, where the work is a
 * server refetch — and shared so they all settle on one window rather than each
 * redeclaring it. It is the hook's default rather than its only value: a
 * consumer wanting a different pause states that delay at the call site.
 */
export const SEARCH_DEBOUNCE_MS = 300;

/**
 * Track `value`, but only publish it once it has held still for `delayMs`.
 *
 * The debounced value starts at the initial `value` rather than at a blank —
 * a list mounted with a search term already in its state queries for that term
 * immediately instead of fetching the unfiltered page first.
 *
 * The pending timer is cleared on every change and on unmount, so a fast typist
 * causes one publish per pause, and an unmounted component never publishes.
 */
export function useDebouncedValue<T>(
  value: T,
  delayMs: number = SEARCH_DEBOUNCE_MS
): T {
  const [debounced, setDebounced] = useState<T>(value);

  useEffect(() => {
    const handle = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(handle);
  }, [value, delayMs]);

  return debounced;
}
