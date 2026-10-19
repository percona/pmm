import Box from '@mui/material/Box';
import {
  LoadableChildren,
  StatusIcon,
  type StatusIconSeverity,
} from '@percona/peak-ui';
import { Fragment, type FC, type ReactNode } from 'react';
import { Messages } from '../hub.messages';
import type { HubStatus } from '../hub.types';
import { DoorLink } from './DoorLink';
import { visuallyHidden } from './hub.sx';
import { formatAge } from './hub.time';

const SEVERITY: Record<HubStatus['state'], StatusIconSeverity> = {
  healthy: 'success',
  degraded: 'warning',
  down: 'error',
  noData: 'indeterminate',
  empty: 'indeterminate',
};

export interface HubStatusLineProps {
  status: HubStatus;
  /** Technology name, such as "MongoDB". */
  name: string;
  /** Singular and plural of the unit counted, such as ["cluster", "clusters"]. */
  unit: [string, string];
  /** Whether the reader can set up a backup, which an estate without any then offers. */
  canSetUpBackup?: boolean;
  /** The environment filter matches no service, so there is nothing to count. */
  matchesNothing?: boolean;
  loading?: boolean;
}

const backupFigure = (status: HubStatus, canSetUp: boolean): ReactNode => {
  const { lastGoodBackup, doors } = status;

  switch (lastGoodBackup) {
    case 'none':
      return (
        <DoorLink door={doors.backup} underline="always">
          {Messages.status.noGoodBackup}
        </DoorLink>
      );
    case 'notSetUp':
      return canSetUp ? (
        <DoorLink door={doors.backup} underline="always">
          {Messages.status.setUp}
        </DoorLink>
      ) : (
        Messages.status.notSetUp
      );
    case 'featureOff':
      return Messages.status.featureOff;
    case 'unknown':
      return Messages.status.backupUnknown;
    default:
      return (
        <DoorLink door={doors.backup} underline="always">
          {Messages.status.lastGoodBackup(formatAge(lastGoodBackup.at))}
        </DoorLink>
      );
  }
};

const attentionFigure = (status: HubStatus): ReactNode => {
  if (status.needAttention == null) {
    return (
      <>
        <span aria-hidden="true">{Messages.status.attentionUnknown}</span>
        <Box component="span" sx={visuallyHidden}>
          {Messages.status.attentionUnknownHidden}
        </Box>
      </>
    );
  }

  if (status.needAttention === 0) {
    return Messages.status.noAttention;
  }

  return (
    <DoorLink door={status.doors.attention} underline="always">
      {Messages.status.attention(status.needAttention)}
    </DoorLink>
  );
};

/** One sentence on the whole estate; each figure opens what it counts. */
export const HubStatusLine: FC<HubStatusLineProps> = ({
  status,
  name,
  unit,
  canSetUpBackup = false,
  matchesNothing = false,
  loading = false,
}) => {
  const figures: ReactNode[] =
    status.state === 'empty'
      ? [Messages.status.empty(name)]
      : matchesNothing
        ? [Messages.environment.nothing(name)]
        : [
            <DoorLink
              key="clusters"
              door={status.doors.clusters}
              underline="always"
            >
              {Messages.status.clusters(status.clusters, unit)}
            </DoorLink>,
            attentionFigure(status),
            backupFigure(status, canSetUpBackup),
          ];

  return (
    <Box
      sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}
      data-testid="hub-status-line"
    >
      <LoadableChildren loading={loading} skeletonProps={{ width: 22 }}>
        <StatusIcon size="small" severity={SEVERITY[status.state]} />
      </LoadableChildren>
      <Box component="span" sx={{ minWidth: 0 }}>
        <LoadableChildren loading={loading} skeletonProps={{ width: 320 }}>
          <span>
            {figures.map((figure, index) => (
              <Fragment key={index}>
                {index > 0 && ' · '}
                {figure}
              </Fragment>
            ))}
          </span>
        </LoadableChildren>
      </Box>
    </Box>
  );
};
