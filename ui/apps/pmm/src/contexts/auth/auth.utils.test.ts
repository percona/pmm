import { afterEach, describe, expect, it } from 'vitest';
import {
  getRefetchInterval,
  getSessionExpiry,
  isSessionExpired,
} from './auth.utils';
import { MIN_ROTATE_DELAY_MS } from './auth.constants';

const nowInSeconds = () => Math.floor(Date.now() / 1000);

const setSessionExpiry = (unixSeconds: number) => {
  document.cookie = `grafana_session_expiry=${unixSeconds}`;
};

const clearCookies = () => {
  for (const cookie of document.cookie.split('; ')) {
    const name = cookie.split('=')[0];

    if (name) {
      document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
    }
  }
};

describe('auth.utils', () => {
  afterEach(() => {
    clearCookies();
  });

  describe('getSessionExpiry', () => {
    it('returns 0 without the cookie', () => {
      expect(getSessionExpiry()).toBe(0);
    });

    it('returns the cookie value', () => {
      setSessionExpiry(1788515211);

      expect(getSessionExpiry()).toBe(1788515211);
    });
  });

  describe('isSessionExpired', () => {
    it('is expired without the cookie', () => {
      expect(isSessionExpired()).toBe(true);
    });

    it('is expired once the deadline has passed', () => {
      setSessionExpiry(nowInSeconds() - 60);

      expect(isSessionExpired()).toBe(true);
    });

    it('is not expired before the deadline', () => {
      setSessionExpiry(nowInSeconds() + 60);

      expect(isSessionExpired()).toBe(false);
    });
  });

  describe('getRefetchInterval', () => {
    // React Query drops a negative interval and then schedules nothing at all, which would
    // leave the session unrotated for the life of the page.
    it('never returns a value below the floor when the deadline has passed', () => {
      setSessionExpiry(nowInSeconds() - 3600);

      expect(getRefetchInterval()).toBe(MIN_ROTATE_DELAY_MS);
    });

    it('never returns a value below the floor without the cookie', () => {
      expect(getRefetchInterval()).toBe(MIN_ROTATE_DELAY_MS);
    });

    it('schedules ahead of a future deadline, minus the distribution window', () => {
      const anHour = 3600;
      setSessionExpiry(nowInSeconds() + anHour);

      const interval = getRefetchInterval();

      expect(interval).toBeGreaterThan(MIN_ROTATE_DELAY_MS);
      expect(interval).toBeLessThanOrEqual(anHour * 1000);
      // 20s of jitter, plus up to 1s of sub-second truncation
      expect(interval).toBeGreaterThanOrEqual(anHour * 1000 - 21000);
    });
  });
});
