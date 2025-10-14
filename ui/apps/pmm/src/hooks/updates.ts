import { useUpdates } from 'contexts/updates';
import { useUser } from 'contexts/user';
import { useSnoozeUpdate } from './api/useUser';
import { useCallback, useMemo } from 'react';
import { useSettings } from 'contexts/settings';
import { parseDuration } from 'utils/duration.utils';
import { diffFromNow } from 'utils/datetime.utils';
import { SHOW_UPDATE_MODAL_AFTER_MS } from 'lib/constants';

export const useSnooze = () => {
  const { settings } = useSettings();
  const { versionInfo } = useUpdates();
  const { user } = useUser();
  const { mutateAsync } = useSnoozeUpdate();
  const latest = versionInfo?.latest || null;
  const snoozeActive = useMemo(() => {
    if (
      latest?.timestamp &&
      diffFromNow(latest.timestamp) < SHOW_UPDATE_MODAL_AFTER_MS
    ) {
      return true;
    }

    if (
      latest?.version !== user?.info.snoozedPmmVersion ||
      !user?.info.snoozedAt ||
      !settings?.updatesSnoozeDuration
    ) {
      return false;
    }

    return (
      diffFromNow(user?.info.snoozedAt) <=
      parseDuration(settings.updatesSnoozeDuration)
    );
  }, [latest, user, settings]);

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
