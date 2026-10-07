/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import type { ReactElement } from 'react';
import { Stack, Tooltip, Typography } from '@mui/material';
import Chip from '@mui/material/Chip';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import {
  CLUSTER_HEALTH_COLOR,
  CLUSTER_HEALTH_LABEL,
  METRICS_LOOKBACK,
  RUN_STATUS_COLOR,
  RUN_STATUS_LABEL,
  SERVICE_STATUS_COLOR,
  SERVICE_STATUS_LABEL,
} from '../constants';
import { isRunActive } from '../api';
import { formatCompactDuration, formatTimestamp } from '../format';
import { ageSeconds } from '../inventory';
import type {
  OmClusterHealth,
  OmServiceStatus,
  OmTopologyRunStatus,
} from '../types';

const SERVICE_STATUS_ICON: Record<OmServiceStatus, ReactElement> = {
  SERVICE_STATUS_UP: <CheckCircleOutlineIcon />,
  SERVICE_STATUS_DOWN: <ErrorOutlineIcon />,
  SERVICE_STATUS_UNSPECIFIED: <HelpOutlineIcon />,
};

const CLUSTER_HEALTH_ICON: Record<OmClusterHealth, ReactElement> = {
  healthy: <CheckCircleOutlineIcon />,
  degraded: <WarningAmberIcon />,
  down: <ErrorOutlineIcon />,
  unknown: <HelpOutlineIcon />,
};

/**
 * Service reachability as a chip.
 *
 * DOWN covers both "the exporter said 0" and "the service produced no metrics at
 * all" — the worker collapses them deliberately, because from the estate's point of
 * view an unreachable service and an unmonitored one are the same problem.
 */
/**
 * Down is the filled chip and Up the outlined one: a failure has to be the loudest
 * thing on the page, and the healthy state the quietest. Each state also carries its
 * own icon and word, so none of them depends on colour to be read.
 */
export const StatusBadge = ({
  status,
  lastUpAt,
}: {
  status: OmServiceStatus;
  /** The service's `last_up_at`, read only when it is down. */
  lastUpAt?: string | null;
}) => {
  // A value this build does not know reads as unknown throughout, so the icon, the
  // word and the colour cannot disagree.
  const known: OmServiceStatus =
    status in SERVICE_STATUS_LABEL ? status : 'SERVICE_STATUS_UNSPECIFIED';
  const chip = (
    <Chip
      size="small"
      icon={SERVICE_STATUS_ICON[known]}
      label={SERVICE_STATUS_LABEL[known]}
      color={SERVICE_STATUS_COLOR[known]}
      variant={known === 'SERVICE_STATUS_DOWN' ? 'filled' : 'outlined'}
      data-testid="om-service-status"
    />
  );
  if (known !== 'SERVICE_STATUS_DOWN') {
    return chip;
  }
  return (
    <Stack direction="row" alignItems="center" gap={0.75}>
      {chip}
      <DownFor lastUpAt={lastUpAt} />
    </Stack>
  );
};

/**
 * How long a down service has been down, beside its chip: "for 3h", the exact time in
 * the tooltip. From the server's `last_up_at`, so it reads the same after a reload.
 */
const DownFor = ({ lastUpAt }: { lastUpAt?: string | null }) => {
  const age = ageSeconds(lastUpAt);
  const [text, title] =
    age == null
      ? [
          `not up in ${METRICS_LOOKBACK}`,
          `No metric shows it up at any point in the last ${METRICS_LOOKBACK}.`,
        ]
      : [
          `for ${formatCompactDuration(age) || '0s'}`,
          `Last seen up ${formatTimestamp(lastUpAt)}`,
        ];
  return (
    <Tooltip title={title}>
      <Typography
        variant="caption"
        color="text.secondary"
        noWrap
        data-testid="om-down-for"
      >
        {text}
      </Typography>
    </Tooltip>
  );
};

/**
 * A cluster's state as a chip. Filled only for the states that need someone: a
 * degraded or down cluster, never a healthy one.
 */
export const ClusterHealthBadge = ({ health }: { health: OmClusterHealth }) => {
  return (
    <Chip
      size="small"
      icon={CLUSTER_HEALTH_ICON[health]}
      label={CLUSTER_HEALTH_LABEL[health]}
      color={CLUSTER_HEALTH_COLOR[health]}
      variant={
        health === 'down' || health === 'degraded' ? 'filled' : 'outlined'
      }
      data-testid="om-cluster-health"
    />
  );
};

/** Discovery-run status as a chip. */
export const RunStatusBadge = ({ status }: { status: OmTopologyRunStatus }) => {
  return (
    <Chip
      size="small"
      label={RUN_STATUS_LABEL[status] ?? status}
      color={RUN_STATUS_COLOR[status] ?? 'default'}
      variant={isRunActive(status) ? 'outlined' : 'filled'}
    />
  );
};
