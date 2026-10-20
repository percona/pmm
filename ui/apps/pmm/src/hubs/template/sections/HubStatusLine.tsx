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
  upgrading: 'info',
  noData: 'indeterminate',
  empty: 'indeterminate',
};

// Worst first: unknown outranks info and success, never a warning.
const ICON_RANK: StatusIconSeverity[] = [
  'error',
  'warning',
  'indeterminate',
  'info',
  'success',
];

/** The worse of the estate's health and its most severe attention row, with the word that names it. */
const statusIcon = ({
  state,
  attentionSeverity,
}: HubStatus): { severity: StatusIconSeverity; word?: string } => {
  const health = SEVERITY[state];

  if (
    attentionSeverity &&
    ICON_RANK.indexOf(attentionSeverity) < ICON_RANK.indexOf(health)
  ) {
    return {
      severity: attentionSeverity,
      word: Messages.attention.severity[attentionSeverity],
    };
  }

  return {
    severity: health,
    word: state === 'empty' ? undefined : Messages.health[state],
  };
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

const otherIssuesFigure = (status: HubStatus): ReactNode[] =>
  status.needAttention != null && status.otherIssues > 0
    ? [
        <DoorLink
          key="otherIssues"
          door={status.doors.attention}
          underline="always"
        >
          {Messages.status.otherIssues(status.otherIssues)}
        </DoorLink>,
      ]
    : [];

const alertsFigure = ({ alerts, doors }: HubStatus): ReactNode[] => {
  if (alerts === undefined) {
    return [];
  }
  if (alerts === null) {
    return [
      <Fragment key="alerts">
        <span aria-hidden="true">{Messages.status.alertsUnknown}</span>
        <Box component="span" sx={visuallyHidden}>
          {Messages.status.alertsUnknownHidden}
        </Box>
      </Fragment>,
    ];
  }

  return [
    alerts > 0 ? (
      <DoorLink key="alerts" door={doors.alerts} underline="always">
        {Messages.status.alertsFiring(alerts)}
      </DoorLink>
    ) : (
      Messages.status.noAlertsFiring
    ),
  ];
};

const patchesFigure = ({ patches, doors }: HubStatus): ReactNode[] =>
  patches
    ? [
        <Fragment key="patches">
          {patches.behind > 0 ? (
            <DoorLink door={doors.patches} underline="always">
              {Messages.status.patchesBehind(patches.behind)}
            </DoorLink>
          ) : (
            Messages.status.patchesNone
          )}
          {patches.unknown > 0 &&
            Messages.status.patchesUnknown(patches.unknown)}
        </Fragment>,
      ]
    : [];

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
            status.clusters == null ? (
              <Fragment key="clusters">
                <span aria-hidden="true">
                  {Messages.status.clustersUnknown(unit)}
                </span>
                <Box component="span" sx={visuallyHidden}>
                  {Messages.status.clustersUnknownHidden(unit)}
                </Box>
              </Fragment>
            ) : (
              <DoorLink
                key="clusters"
                door={status.doors.clusters}
                underline="always"
              >
                {Messages.status.clusters(status.clusters, unit)}
              </DoorLink>
            ),
            attentionFigure(status),
            ...otherIssuesFigure(status),
            ...alertsFigure(status),
            ...patchesFigure(status),
            backupFigure(status, canSetUpBackup),
          ];

  const icon = statusIcon(status);

  return (
    <Box
      sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}
      data-testid="hub-status-line"
    >
      <LoadableChildren loading={loading} skeletonProps={{ width: 22 }}>
        <StatusIcon
          size="small"
          severity={icon.severity}
          // The sentence says it for an empty estate or an unmatched filter; elsewhere only the icon does.
          titleAccess={matchesNothing ? undefined : icon.word}
        />
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
