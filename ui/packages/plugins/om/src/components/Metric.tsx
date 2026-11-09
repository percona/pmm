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

import { formatCompactDuration } from '../format';
import { Unavailable } from './Unavailable';

/** Reading that means "not measured" in the two percentage columns. */
export const UNMEASURED = -1;

/**
 * Render a percentage, or the reason there is none.
 *
 * `-1` is the worker's not-measured sentinel; every other value — including `0` —
 * is a real reading and must render as a number. A falsy check here would report an
 * idle server as unmonitored.
 */
export const Percent = ({ value }: { value: number }) => {
  if (value === UNMEASURED) {
    return <Unavailable reason="metric_not_collected" />;
  }
  return <>{value.toFixed(1)}%</>;
};

/**
 * Render a duration, or the reason there is none.
 *
 * Null here means the field does not apply to this topology — a router has no oplog —
 * which is a different statement from the percentage columns' `-1`.
 */
export const Duration = ({ value }: { value: number | null | undefined }) => {
  if (value === null || value === undefined) {
    return <Unavailable reason="not_applicable" />;
  }
  return <>{formatCompactDuration(value) || '0s'}</>;
};
