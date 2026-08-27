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

import { describe, expect, it } from 'vitest';
import {
  formatAge,
  formatCompactDuration,
  formatRunDuration,
  formatTimestamp,
} from '../src/format';

describe('formatCompactDuration', () => {
  it('renders sub-minute values in seconds', () => {
    expect(formatCompactDuration(0)).toBe('0s');
    expect(formatCompactDuration(47)).toBe('47s');
  });

  it('stops at two units so a value stays glanceable', () => {
    expect(formatCompactDuration(47323)).toBe('13h 8m');
    expect(formatCompactDuration(90)).toBe('1m 30s');
    expect(formatCompactDuration(2 * 86400 + 3 * 3600 + 7 * 60)).toBe('2d 3h');
  });

  it('skips units that are zero rather than padding them', () => {
    expect(formatCompactDuration(86400)).toBe('1d');
    expect(formatCompactDuration(3600)).toBe('1h');
  });

  // A null here means "not observed", and the caller renders <Unavailable/>.
  // Returning '0s' would turn an absent observation into a real measurement.
  it('returns empty for values that are not measurements', () => {
    expect(formatCompactDuration(null)).toBe('');
    expect(formatCompactDuration(undefined)).toBe('');
    expect(formatCompactDuration(-1)).toBe('');
    expect(formatCompactDuration(Number.NaN)).toBe('');
  });
});

describe('formatAge', () => {
  const now = Date.parse('2026-08-07T12:00:00Z');

  it('renders the age of a timestamp', () => {
    expect(formatAge('2026-08-07T11:57:00Z', now)).toBe('3m ago');
  });

  it('clamps a future timestamp to zero rather than going negative', () => {
    expect(formatAge('2026-08-07T12:05:00Z', now)).toBe('0s ago');
  });

  it('renders an em-dash when there is no timestamp', () => {
    expect(formatAge(null, now)).toBe('—');
    expect(formatAge('not-a-date', now)).toBe('—');
  });
});

describe('formatTimestamp', () => {
  it('renders an em-dash for a missing or unparsable value', () => {
    expect(formatTimestamp(null)).toBe('—');
    expect(formatTimestamp('nonsense')).toBe('—');
  });

  // A fixed pattern rather than toLocaleString: run timestamps are compared against
  // one another down a column, and a format that varies with the reader's locale is
  // not a column you can scan. The rendering is local time, so the assertion goes
  // through the same conversion rather than hard-coding a zone.
  it('renders a parsable timestamp in local time', () => {
    const iso = '2026-08-07T09:58:48Z';
    const local = new Date(iso);
    const pad = (value: number) => String(value).padStart(2, '0');
    expect(formatTimestamp(iso)).toBe(
      `${local.getFullYear()}-${pad(local.getMonth() + 1)}-${pad(local.getDate())} ` +
        `${pad(local.getHours())}:${pad(local.getMinutes())}:${pad(local.getSeconds())}`
    );
  });
});

describe('formatRunDuration', () => {
  it('measures a finished run', () => {
    expect(
      formatRunDuration('2026-08-07T09:58:48Z', '2026-08-07T09:59:27Z')
    ).toBe('39s');
  });

  // A run still going has no duration yet; the column shows a placeholder
  // rather than counting up from a start it cannot bound.
  it('is empty while a run is still in flight', () => {
    expect(formatRunDuration('2026-08-07T09:58:48Z', null)).toBe('');
  });

  it('is empty when the timestamps are inconsistent', () => {
    expect(
      formatRunDuration('2026-08-07T09:59:27Z', '2026-08-07T09:58:48Z')
    ).toBe('');
  });
});
