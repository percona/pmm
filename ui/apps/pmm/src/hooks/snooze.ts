import { useUpdates } from 'contexts/updates';
import { useUpdateUserInfo } from './api/useUser';
import { useUser } from 'contexts/user';

export const useSnooze = () => {
  const { versionInfo } = useUpdates();
  const { user } = useUser();
  const { mutateAsync } = useUpdateUserInfo();
  const installedVersion = versionInfo?.installed.version || null;
  const latestVersion = versionInfo?.latest.version || null;

  const snoozeUpdate = async () => {
    if (!latestVersion || !installedVersion || !user?.info) {
      return;
    }

    if (latestVersion !== user.info.snoozedPmmVersion) {
      await mutateAsync({
        snoozedCount: 1,
        snoozedAt: new Date().toISOString(),
        snoozedPmmVersion: latestVersion,
      });
    } else {
      await mutateAsync({
        snoozedCount: user.info.snoozedCount + 1,
        snoozedAt: new Date().toISOString(),
      });
    }
  };

  return { snoozeUpdate, snoozeCount: user?.info.snoozedCount || 0 };
};
