import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import {
  DEFAULT_UPDATE_SNOOZE_DURATION_MS,
  SHOW_UPDATE_MODAL_AFTER_MS,
  UPDATE_SNOOZE_DURATION_OVERRIDE_KEY,
} from 'lib/constants';
import type { LatestInfo } from 'types/updates.types';
import {
  getUpdateSnoozeDurationMs,
  isUpdateSnoozeActive,
} from './updates.utils';

const NOW = new Date('2026-07-24T12:00:00.000Z').getTime();
const OLD_RELEASE_TIMESTAMP = new Date(
  NOW - 2 * SHOW_UPDATE_MODAL_AFTER_MS
).toISOString();

const latestVersion = (overrides: Partial<LatestInfo> = {}): LatestInfo => ({
  version: '3.1.0',
  tag: '',
  timestamp: OLD_RELEASE_TIMESTAMP,
  releaseNotesText: '',
  releaseNotesUrl: '',
  ...overrides,
});

describe('isUpdateSnoozeActive', () => {
  beforeEach(() => {
    localStorage.removeItem(UPDATE_SNOOZE_DURATION_OVERRIDE_KEY);
  });

  afterEach(() => {
    localStorage.removeItem(UPDATE_SNOOZE_DURATION_OVERRIDE_KEY);
  });

  it('returns true when latest is missing so the popup does not flash', () => {
    expect(
      isUpdateSnoozeActive(
        null,
        {
          snoozedAt: null,
          snoozedPmmVersion: '',
        },
        NOW
      )
    ).toBe(true);
  });

  it('returns true when user info is missing so the popup does not flash', () => {
    expect(isUpdateSnoozeActive(latestVersion(), null, NOW)).toBe(true);
  });

  it('returns false when the user has never snoozed and release is older than one hour', () => {
    expect(
      isUpdateSnoozeActive(
        latestVersion(),
        {
          snoozedAt: null,
          snoozedPmmVersion: '',
        },
        NOW
      )
    ).toBe(false);
  });

  it('returns true when snoozed for the current version within the 7-day window', () => {
    expect(
      isUpdateSnoozeActive(
        latestVersion(),
        {
          snoozedPmmVersion: '3.1.0',
          snoozedAt: new Date(NOW - 60_000).toISOString(),
        },
        NOW
      )
    ).toBe(true);
  });

  it('returns false when the 7-day snooze window has elapsed', () => {
    expect(
      isUpdateSnoozeActive(
        latestVersion(),
        {
          snoozedPmmVersion: '3.1.0',
          snoozedAt: new Date(
            NOW - DEFAULT_UPDATE_SNOOZE_DURATION_MS - 60_000
          ).toISOString(),
        },
        NOW
      )
    ).toBe(false);
  });

  it('returns false when snoozed for a different version', () => {
    expect(
      isUpdateSnoozeActive(
        latestVersion({ version: '3.2.0' }),
        {
          snoozedPmmVersion: '3.1.0',
          snoozedAt: new Date(NOW - 60_000).toISOString(),
        },
        NOW
      )
    ).toBe(false);
  });

  it('returns true when the latest release is younger than one hour', () => {
    expect(
      isUpdateSnoozeActive(
        latestVersion({
          timestamp: new Date(NOW - 60_000).toISOString(),
        }),
        {
          snoozedAt: null,
          snoozedPmmVersion: '',
        },
        NOW
      )
    ).toBe(true);
  });

  it('returns true when localStorage override is set and snooze is still within that window', () => {
    localStorage.setItem(UPDATE_SNOOZE_DURATION_OVERRIDE_KEY, '10000');

    expect(
      isUpdateSnoozeActive(
        latestVersion(),
        {
          snoozedPmmVersion: '3.1.0',
          snoozedAt: new Date(NOW - 5_000).toISOString(),
        },
        NOW
      )
    ).toBe(true);
  });

  it('returns false when localStorage override is set and snooze has expired', () => {
    localStorage.setItem(UPDATE_SNOOZE_DURATION_OVERRIDE_KEY, '10000');

    expect(
      isUpdateSnoozeActive(
        latestVersion(),
        {
          snoozedPmmVersion: '3.1.0',
          snoozedAt: new Date(NOW - 15_000).toISOString(),
        },
        NOW
      )
    ).toBe(false);
  });
});

describe('getUpdateSnoozeDurationMs', () => {
  beforeEach(() => {
    localStorage.removeItem(UPDATE_SNOOZE_DURATION_OVERRIDE_KEY);
  });

  afterEach(() => {
    localStorage.removeItem(UPDATE_SNOOZE_DURATION_OVERRIDE_KEY);
  });

  it('returns the default duration when no override is set', () => {
    expect(getUpdateSnoozeDurationMs()).toBe(DEFAULT_UPDATE_SNOOZE_DURATION_MS);
  });

  it('returns the override when a valid positive duration is set', () => {
    localStorage.setItem(UPDATE_SNOOZE_DURATION_OVERRIDE_KEY, '10000');
    expect(getUpdateSnoozeDurationMs()).toBe(10000);
  });

  it.each(['', 'abc', '0', '-1'])(
    'falls back to the default duration for invalid override %j',
    (value) => {
      localStorage.setItem(UPDATE_SNOOZE_DURATION_OVERRIDE_KEY, value);
      expect(getUpdateSnoozeDurationMs()).toBe(
        DEFAULT_UPDATE_SNOOZE_DURATION_MS
      );
    }
  );
});
