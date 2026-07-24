import {
  DEFAULT_UPDATE_SNOOZE_DURATION_MS,
  SHOW_UPDATE_MODAL_AFTER_MS,
} from 'lib/constants';
import { LatestInfo } from 'types/updates.types';
import { UserInfo } from 'types/user.types';

type SnoozeUserInfo = Pick<UserInfo, 'snoozedAt' | 'snoozedPmmVersion'>;

export const isUpdateSnoozeActive = ({
  latest,
  userInfo,
  now = Date.now(),
}: {
  latest: LatestInfo | null;
  userInfo: SnoozeUserInfo | null;
  now?: number;
}): boolean => {
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
    now - new Date(userInfo.snoozedAt).getTime() <=
    DEFAULT_UPDATE_SNOOZE_DURATION_MS
  );
};
