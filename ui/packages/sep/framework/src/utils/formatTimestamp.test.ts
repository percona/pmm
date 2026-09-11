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
  formatAbsoluteTime,
  formatRelativeTime,
  formatTimestamp,
} from './formatTimestamp';

const NOW = new Date('2026-06-18T12:00:00Z').getTime();

describe('formatRelativeTime', () => {
  it('formats a future time', () => {
    expect(formatRelativeTime('2026-06-18T14:00:00Z', NOW)).toBe('in 2 hours');
  });

  it('formats a past time', () => {
    expect(formatRelativeTime('2026-06-15T12:00:00Z', NOW)).toBe('3 days ago');
  });

  it('returns a non-date unchanged', () => {
    expect(formatRelativeTime('not-a-date', NOW)).toBe('not-a-date');
  });
});

describe('formatAbsoluteTime', () => {
  it('renders an em-dash for an absent value', () => {
    expect(formatAbsoluteTime(null)).toBe('—');
    expect(formatAbsoluteTime(undefined)).toBe('—');
    expect(formatAbsoluteTime('')).toBe('—');
  });

  it('returns a non-date unchanged', () => {
    expect(formatAbsoluteTime('not-a-date')).toBe('not-a-date');
  });
});

describe('formatTimestamp', () => {
  it('returns null for an absent value so the cell renders empty', () => {
    expect(formatTimestamp(null, NOW)).toBeNull();
    expect(formatTimestamp(undefined, NOW)).toBeNull();
    expect(formatTimestamp('', NOW)).toBeNull();
  });

  it('shows a recent past time as relative', () => {
    expect(formatTimestamp('2026-06-15T12:00:00Z', NOW)?.display).toBe(
      '3 days ago'
    );
  });

  it('shows a near-future time as relative', () => {
    expect(formatTimestamp('2026-06-18T14:00:00Z', NOW)?.display).toBe(
      'in 2 hours'
    );
  });

  it('falls back to absolute once past the relative window', () => {
    const old = formatTimestamp('2025-01-02T03:04:00Z', NOW);
    expect(old?.display).not.toMatch(/ago/);
    expect(old?.display).toBe(formatAbsoluteTime('2025-01-02T03:04:00Z'));
  });

  it('falls back to absolute for a far-future time', () => {
    const far = formatTimestamp('2027-01-02T03:04:00Z', NOW);
    expect(far?.display).not.toMatch(/^in /);
    expect(far?.display).toBe(formatAbsoluteTime('2027-01-02T03:04:00Z'));
  });

  it('always offers the full timestamp for hover, including when absolute', () => {
    const recent = formatTimestamp('2026-06-17T12:00:00Z', NOW);
    expect(recent?.display).toBe('yesterday');
    expect(recent?.title).toBe(formatAbsoluteTime('2026-06-17T12:00:00Z'));

    const old = formatTimestamp('2025-01-02T03:04:00Z', NOW);
    expect(old?.title).toBe(formatAbsoluteTime('2025-01-02T03:04:00Z'));
  });

  it('keeps the boundary inclusive on the relative side', () => {
    // A hair under seven days stays relative; a hair over crosses to absolute.
    const justInside = NOW - (7 * 24 * 60 * 60 * 1000 - 1000);
    const justOutside = NOW - (7 * 24 * 60 * 60 * 1000 + 1000);
    expect(
      formatTimestamp(new Date(justInside).toISOString(), NOW)?.display
    ).toMatch(/ago/);
    expect(
      formatTimestamp(new Date(justOutside).toISOString(), NOW)?.display
    ).not.toMatch(/ago/);
  });

  it('shows a malformed value rather than Invalid Date', () => {
    expect(formatTimestamp('not-a-date', NOW)).toEqual({
      display: 'not-a-date',
      title: 'not-a-date',
    });
  });
});
