import { useUpdates } from 'contexts/updates';
import { useUser } from 'contexts/user';
import { useSnoozeUpdate } from './api/useUpdates';
import { useMemo } from 'react';
import { useSettings } from 'contexts/settings';
import { parseDuration } from 'utils/duration';

export const useSnooze = () => {
  const { settings } = useSettings();
  const { versionInfo } = useUpdates();
  const { user } = useUser();
  const { mutateAsync } = useSnoozeUpdate();
  const latestVersion = versionInfo?.latest.version || null;
  const snoozeActive = useMemo(() => {
    if (latestVersion !== user?.info.snoozedPmmVersion) {
      return false;
    }

    return user?.info.snoozedAt && settings?.updatesSnoozeDuration
      ? new Date().getTime() - new Date(user?.info.snoozedAt).getTime() <=
          parseDuration(settings.updatesSnoozeDuration)
      : false;
  }, [latestVersion, user?.info.snoozedAt, settings?.updatesSnoozeDuration]);

  const snoozeUpdate = async () => {
    if (!latestVersion) {
      return;
    }

    await mutateAsync({
      snoozedPmmVersion: latestVersion,
    });
  };

  return {
    snoozeUpdate,
    snoozeActive,
    snoozeCount: user?.info.snoozedCount || 0,
    snoozedAt: user?.info.snoozedAt || null,
  };
};
