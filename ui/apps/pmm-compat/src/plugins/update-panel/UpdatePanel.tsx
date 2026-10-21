import React, { useState, type FC, type MouseEvent } from 'react';
import { config, locationService } from '@grafana/runtime';
import { Button, Spinner, Tooltip } from '@grafana/ui';
import { useCheckUpdates, useReadonlySettings } from 'plugins/shared/hooks';
import { isPmmAdmin } from 'plugins/shared/utils/permissions.utils';
import { AvailableUpdate } from './components/available-update';
import { CurrentVersion } from './components/current-version';
import { DeprecationTooltip } from './components/deprecation-tooltip';
import { InfoBox } from './components/info-box';
import { LastCheck } from './components/last-check';
import { PMM_UPDATES_URL } from './UpdatePanel.constants';
import { Messages } from './UpdatePanel.messages';
import { styles } from './UpdatePanel.styles';
import { formatDateWithTime, isPmmNavEnabled } from './UpdatePanel.utils';

export const UpdatePanel: FC = () => {
  const isOnline = navigator.onLine;
  const isAuthorized = isPmmAdmin(config.bootData.user);
  const { data: settings, isLoading: isLoadingSettings } =
    useReadonlySettings();
  const {
    data: updates,
    isFetching: isLoadingVersionDetails,
    refetch,
  } = useCheckUpdates(isAuthorized && !!settings?.updatesEnabled);
  const [forceUpdate, setForceUpdate] = useState(false);
  const { installed, latest, latestNewsUrl, updateAvailable, lastChecked } =
    updates ?? {};
  const isDefaultView = !latest;
  const isLoading = isLoadingVersionDetails || isLoadingSettings;

  const handleCheckForUpdates = (e: MouseEvent) => {
    if (e.altKey) {
      setForceUpdate(true);
    }

    refetch();
  };

  const handleOpenUpdates = () => {
    if (isPmmNavEnabled()) {
      locationService.push(PMM_UPDATES_URL);
    } else {
      window.location.assign(PMM_UPDATES_URL);
    }
  };

  return (
    <div className={styles.panel}>
      {!!installed && <CurrentVersion currentVersion={installed} />}
      {updateAvailable &&
      !isDefaultView &&
      settings?.updatesEnabled &&
      isAuthorized &&
      !isLoading &&
      isOnline ? (
        <AvailableUpdate nextVersion={latest} newsLink={latestNewsUrl} />
      ) : null}
      {isLoading ? (
        <div className={styles.middleSectionWrapper}>
          <Spinner />
        </div>
      ) : (updateAvailable || forceUpdate) &&
        settings?.updatesEnabled &&
        isAuthorized &&
        isOnline ? (
        <div className={styles.middleSectionWrapper}>
          <Tooltip interactive content={<DeprecationTooltip />}>
            <Button
              onClick={handleOpenUpdates}
              variant="secondary"
              icon="exclamation-triangle"
            >
              {latest?.version
                ? Messages.upgradeTo(latest.version)
                : Messages.upgrade}
            </Button>
          </Tooltip>
        </div>
      ) : (
        <InfoBox
          upToDate={!isDefaultView && !forceUpdate}
          hasNoAccess={!isAuthorized}
          updatesDisabled={!settings?.updatesEnabled}
          isOnline={isOnline}
        />
      )}
      <LastCheck
        disabled={isLoading || !settings?.updatesEnabled || !isOnline}
        onCheckForUpdates={handleCheckForUpdates}
        lastCheckDate={lastChecked ? formatDateWithTime(lastChecked) : ''}
      />
    </div>
  );
};
