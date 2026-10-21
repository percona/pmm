import React, { useEffect, type FC } from 'react';
import { cx } from '@emotion/css';
import type { PanelProps } from '@grafana/data';
import { config, RefreshEvent } from '@grafana/runtime';
import { Spinner, Tooltip, useStyles2 } from '@grafana/ui';
import { useFailedChecks, useReadonlySettings } from 'plugins/shared/hooks';
import { isEditor, isPmmAdmin } from 'plugins/shared/utils/permissions.utils';
import {
  PMM_DATABASE_CHECKS_PANEL_URL,
  PMM_SETTINGS_URL,
  REFRESH_DEBOUNCE_MS,
} from '../../CheckPanel.constants';
import { splitSeverities, sumSeverities } from '../../CheckPanel.utils';
import { TooltipText } from '../tooltip-text';
import { Messages } from './Failed.messages';
import { getStyles } from './Failed.styles';

export const Failed: FC<PanelProps> = ({ eventBus }) => {
  const styles = useStyles2(getStyles);
  const { user } = config.bootData;
  const canSeeChecks = isPmmAdmin(user) || isEditor(user);
  const { data: settings, isLoading: settingsLoading } = useReadonlySettings();
  const { data: failedChecks, refetch } = useFailedChecks(canSeeChecks);
  const counts = splitSeverities(failedChecks);
  const { emergency, critical, alert, error, warning, debug, info, notice } =
    counts;
  const sum = sumSeverities(counts);

  useEffect(() => {
    if (!canSeeChecks) {
      return;
    }

    let timeout: ReturnType<typeof setTimeout> | undefined;
    const subscription = eventBus.getStream(RefreshEvent).subscribe(() => {
      clearTimeout(timeout);
      timeout = setTimeout(() => refetch(), REFRESH_DEBOUNCE_MS);
    });

    return () => {
      clearTimeout(timeout);
      subscription.unsubscribe();
    };
  }, [eventBus, refetch, canSeeChecks]);

  if (settingsLoading) {
    return <Spinner />;
  }

  if (!canSeeChecks) {
    return (
      <div className={styles.empty} data-testid="unauthorized">
        {Messages.insufficientPermissions}
      </div>
    );
  }

  if (!settings?.advisorEnabled) {
    return (
      <div className={styles.empty} data-testid="db-check-panel-settings-link">
        {Messages.featureDisabled}
        <br />
        {Messages.check}
        <a className={styles.link} href={PMM_SETTINGS_URL}>
          {Messages.pmmSettings}
        </a>
      </div>
    );
  }

  if (!sum) {
    return (
      <div data-testid="db-check-panel-zero-checks">
        <span className={cx(styles.failedDiv, styles.green)}>{sum}</span>
      </div>
    );
  }

  return (
    <div data-testid="db-check-panel-has-checks">
      <Tooltip
        placement="top"
        interactive
        content={<TooltipText counts={counts} />}
      >
        <a href={PMM_DATABASE_CHECKS_PANEL_URL} className={styles.failedDiv}>
          <span
            className={styles.critical}
            data-testid="db-check-panel-critical"
          >
            {emergency + alert + critical}
          </span>
          <span> / </span>
          <span className={styles.error} data-testid="db-check-panel-error">
            {error}
          </span>
          <span> / </span>
          <span className={styles.warning} data-testid="db-check-panel-warning">
            {warning}
          </span>
          <span> / </span>
          <span className={styles.notice} data-testid="db-check-panel-notice">
            {notice + info + debug}
          </span>
        </a>
      </Tooltip>
    </div>
  );
};
