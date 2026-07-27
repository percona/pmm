import { describe, expect, it } from 'vitest';
import {
  DEFAULT_UPDATE_SNOOZE_DURATION_MS,
  SHOW_UPDATE_MODAL_AFTER_MS,
} from 'lib/constants';
import { LatestInfo } from 'types/updates.types';
import { isUpdateSnoozeActive } from './updates.utils';

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
  it('returns true when latest is missing so the popup does not flash', () => {
    expect(
      isUpdateSnoozeActive(null, {
        snoozedAt: null,
        snoozedPmmVersion: '',
      }, NOW)
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
});
