import React, { type FC } from 'react';
import { useStyles2 } from '@grafana/ui';
import {
  DOCKER_UPGRADE_URL,
  HELM_UPGRADE_URL,
  PODMAN_UPGRADE_URL,
} from './DeprecationTooltip.constants';
import { Messages } from './DeprecationTooltip.messages';
import { getStyles } from './DeprecationTooltip.styles';

export const DeprecationTooltip: FC = () => {
  const styles = useStyles2(getStyles);

  return (
    <>
      <strong>{Messages.title}</strong>
      {Messages.description}
      <br />
      {Messages.use}&nbsp;
      <a
        className={styles.link}
        href={DOCKER_UPGRADE_URL}
        target="_blank"
        rel="noreferrer"
      >
        {Messages.docker}
      </a>{' '}
      {Messages.recommended}{' '}
      <a
        className={styles.link}
        href={PODMAN_UPGRADE_URL}
        target="_blank"
        rel="noreferrer"
      >
        {Messages.podman}
      </a>
      {Messages.or}{' '}
      <a
        className={styles.link}
        href={HELM_UPGRADE_URL}
        target="_blank"
        rel="noreferrer"
      >
        {Messages.helm}
      </a>{' '}
      {Messages.futureUpgrades}
    </>
  );
};
