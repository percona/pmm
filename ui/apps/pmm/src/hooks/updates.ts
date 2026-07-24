import { useUpdates } from 'contexts/updates';
import { useUser } from 'contexts/user';
import { useUpdateUserInfo } from './api/useUser';
import { useCallback, useMemo } from 'react';
import { isUpdateSnoozeActive } from './updates.utils';

export const useSnooze = () => {
  const { versionInfo } = useUpdates();
  const { user } = useUser();
  const { mutateAsync } = useUpdateUserInfo();
  const latest = versionInfo?.latest || null;
  const snoozeActive = useMemo(
    () =>
      isUpdateSnoozeActive({
        latest,
        userInfo: user?.info ?? null,
      }),
    [latest, user]
  );

  const snoozeUpdate = useCallback(async () => {
    if (!latest) {
      return;
    }

    await mutateAsync({
      snoozedPmmVersion: latest.version,
    });
  }, [mutateAsync, latest]);

  return {
    snoozeUpdate,
    snoozeActive,
    snoozeCount: user?.info.snoozeCount || 0,
    snoozedAt: user?.info.snoozedAt || null,
  };
};
