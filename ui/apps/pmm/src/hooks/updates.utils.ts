import {
  DEFAULT_UPDATE_SNOOZE_DURATION_MS,
  SHOW_UPDATE_MODAL_AFTER_MS,
  UPDATE_SNOOZE_DURATION_OVERRIDE_KEY,
} from 'lib/constants';
import type { LatestInfo } from 'types/updates.types';
import type { UserInfo } from 'types/user.types';

type SnoozeUserInfo = Pick<UserInfo, 'snoozedAt' | 'snoozedPmmVersion'>;

export const getUpdateSnoozeDurationMs = (): number => {
  try {
    const raw = localStorage.getItem(UPDATE_SNOOZE_DURATION_OVERRIDE_KEY);
    if (raw == null) {
      return DEFAULT_UPDATE_SNOOZE_DURATION_MS;
    }

    const parsed = Number(raw);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return DEFAULT_UPDATE_SNOOZE_DURATION_MS;
    }

    return parsed;
  } catch {
    return DEFAULT_UPDATE_SNOOZE_DURATION_MS;
  }
};

export const isUpdateSnoozeActive = (
  latest: LatestInfo | null,
  userInfo: SnoozeUserInfo | null,
  now: number = Date.now()
): boolean => {
  if (!latest || !userInfo) {
    return true;
  }

  if (
    latest.timestamp &&
    now - new Date(latest.timestamp).getTime() < SHOW_UPDATE_MODAL_AFTER_MS
  ) {
    return true;
  }

  if (latest.version !== userInfo.snoozedPmmVersion || !userInfo.snoozedAt) {
    return false;
  }

  return (
    now - new Date(userInfo.snoozedAt).getTime() <= getUpdateSnoozeDurationMs()
  );
};
