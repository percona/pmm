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

import { keyframes } from '@emotion/react';
import AutorenewIcon from '@mui/icons-material/Autorenew';
import CancelIcon from '@mui/icons-material/Cancel';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CloudOffIcon from '@mui/icons-material/CloudOff';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import HourglassEmptyIcon from '@mui/icons-material/HourglassEmpty';
import HourglassDisabledIcon from '@mui/icons-material/HourglassDisabled';
import ReportIcon from '@mui/icons-material/Report';
import Chip from '@mui/material/Chip';
import type { TaskHistoryStatus } from '../../hooks/useTaskHistory';

const spin = keyframes`
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
`;

const spinningIconSx = {
  '& .MuiChip-icon': {
    animation: `${spin} 1.4s linear infinite`,
  },
} as const;

type ChipColor = 'success' | 'error' | 'warning' | 'info' | 'default';

interface StatusEntry {
  label: string;
  color: ChipColor;
  icon: React.ReactElement;
  spin?: boolean;
}

const STATUS_MAP: Record<TaskHistoryStatus, StatusEntry> = {
  success: { label: 'Done', color: 'success', icon: <CheckCircleIcon /> },
  failed: { label: 'Failed', color: 'error', icon: <ReportIcon /> },
  running: {
    label: 'Running',
    color: 'info',
    icon: <AutorenewIcon />,
    spin: true,
  },
  pending: { label: 'Pending', color: 'default', icon: <HourglassEmptyIcon /> },
  stopped: { label: 'Stopped', color: 'warning', icon: <CancelIcon /> },
  lost: { label: 'Lost', color: 'default', icon: <HelpOutlineIcon /> },
  stale: { label: 'Stale', color: 'default', icon: <HourglassDisabledIcon /> },
  unlaunchable: {
    label: 'Not in executor',
    color: 'warning',
    icon: <CloudOffIcon />,
  },
};

/**
 * Narrow an arbitrary value to a {@link TaskHistoryStatus}.
 *
 * `STATUS_MAP` is the source of truth for the recognized statuses, so call
 * sites that receive loosely-typed values (`unknown` list cells, `task.status`
 * guarded only by `typeof === 'string'`) can validate before handing the value
 * to {@link StatusBadge}, which would otherwise index `STATUS_MAP` with an
 * unrecognized key.
 */
export function isTaskHistoryStatus(
  value: unknown
): value is TaskHistoryStatus {
  return (
    typeof value === 'string' &&
    Object.prototype.hasOwnProperty.call(STATUS_MAP, value)
  );
}

export interface StatusBadgeProps {
  status: TaskHistoryStatus;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const entry = STATUS_MAP[status];
  return (
    <Chip
      size="small"
      color={entry.color}
      icon={entry.icon}
      label={entry.label}
      data-status={status}
      sx={entry.spin ? spinningIconSx : undefined}
    />
  );
}
