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
  INTERVAL_TIMEZONE,
  TIMEZONES,
  browserTimezone,
  defaultPickerTimezone,
  scheduleTimezone,
  utcInputToIso,
  utcIsoToUtcInput,
} from './timezones';
import type { PeriodicTaskResponse } from './hooks';

function task(overrides: Partial<PeriodicTaskResponse>): PeriodicTaskResponse {
  return overrides as PeriodicTaskResponse;
}

describe('utcIsoToUtcInput', () => {
  // The assertions below are absolute on purpose: these helpers read and write
  // UTC parts only, so they hold whatever zone the test runner sits in. A
  // regression back to local-time conversion fails them everywhere except UTC.
  it('writes the instant UTC wall clock, with no shift into the local zone', () => {
    expect(utcIsoToUtcInput('2026-03-01T02:30:00Z')).toBe('2026-03-01T02:30');
  });

  it('keeps a UTC date that local time would push onto another day', () => {
    expect(utcIsoToUtcInput('2026-03-01T23:45:00Z')).toBe('2026-03-01T23:45');
    expect(utcIsoToUtcInput('2026-03-01T00:15:00Z')).toBe('2026-03-01T00:15');
  });

  it('pads single-digit months, days, hours and minutes', () => {
    expect(utcIsoToUtcInput('2026-01-02T03:04:00Z')).toBe('2026-01-02T03:04');
  });

  it('returns an empty string for an unparseable value', () => {
    expect(utcIsoToUtcInput('not-a-date')).toBe('');
  });
});

describe('utcInputToIso', () => {
  it('reads the field back as UTC, not local time', () => {
    expect(utcInputToIso('2026-03-01T02:30')).toBe('2026-03-01T02:30:00.000Z');
  });

  it('accepts a value that already carries seconds', () => {
    expect(utcInputToIso('2026-03-01T02:30:45')).toBe(
      '2026-03-01T02:30:45.000Z'
    );
  });

  it('round-trips with utcIsoToUtcInput', () => {
    const iso = '2026-07-04T18:05:00.000Z';
    expect(utcInputToIso(utcIsoToUtcInput(iso))).toBe(iso);
  });

  it('returns null for a blank or unparseable field', () => {
    expect(utcInputToIso('')).toBeNull();
    expect(utcInputToIso('not-a-date')).toBeNull();
  });
});

describe('scheduleTimezone', () => {
  it("reports the backend's computed zone", () => {
    expect(scheduleTimezone(task({ timezone: 'Europe/Lisbon' }))).toBe(
      'Europe/Lisbon'
    );
  });

  it("falls back to the crontab's own zone when the computed field is absent", () => {
    expect(
      scheduleTimezone(
        task({
          timezone: undefined as unknown as string,
          crontab: {
            minute: '0',
            hour: '2',
            day_of_month: '*',
            month_of_year: '*',
            day_of_week: '*',
            timezone: 'America/New_York',
          },
        })
      )
    ).toBe('America/New_York');
  });

  it('falls back to UTC when nothing names a zone', () => {
    expect(scheduleTimezone(task({}))).toBe(INTERVAL_TIMEZONE);
  });
});

describe('timezone pickers', () => {
  it('offers a non-empty option list including UTC', () => {
    expect(TIMEZONES.length).toBeGreaterThan(0);
    expect(TIMEZONES).toContain('UTC');
  });

  it('defaults the picker to a zone it can actually list', () => {
    expect(TIMEZONES).toContain(defaultPickerTimezone());
  });

  it('reports a browser zone', () => {
    expect(browserTimezone()).toBeTruthy();
  });
});
