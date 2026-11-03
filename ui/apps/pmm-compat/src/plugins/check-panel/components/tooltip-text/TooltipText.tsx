import React, { type FC } from 'react';
import { useStyles2 } from '@grafana/ui';
import { sumSeverities } from '../../CheckPanel.utils';
import { Messages } from './TooltipText.messages';
import { getStyles } from './TooltipText.styles';
import type { TooltipTextProps } from './TooltipText.types';

export const TooltipText: FC<TooltipTextProps> = ({ counts }) => {
  const styles = useStyles2(getStyles);
  const { emergency, critical, alert, error, warning, debug, info, notice } =
    counts;
  const sum = sumSeverities(counts);

  if (!sum) {
    return null;
  }

  return (
    <div className={styles.wrapper}>
      <div className={styles.header}>
        {Messages.failedChecks}&nbsp;{sum}
      </div>
      <div className={styles.body} data-testid="checks-tooltip-body">
        <div>
          {Messages.emergency} &ndash;&nbsp;{emergency}
        </div>
        <div>
          {Messages.alert} &ndash;&nbsp;{alert}
        </div>
        <div>
          {Messages.critical} &ndash;&nbsp;{critical}
        </div>
        <div>
          {Messages.error} &ndash;&nbsp;{error}
        </div>
        <div>
          {Messages.warning} &ndash;&nbsp;{warning}
        </div>
        <div>
          {Messages.notice} &ndash;&nbsp;{notice}
        </div>
        <div>
          {Messages.info} &ndash;&nbsp;{info}
        </div>
        <div>
          {Messages.debug} &ndash;&nbsp;{debug}
        </div>
      </div>
    </div>
  );
};
