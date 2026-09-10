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

import CancelIcon from '@mui/icons-material/Cancel';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CloudOffIcon from '@mui/icons-material/CloudOff';
import ErrorIcon from '@mui/icons-material/Error';
import HelpIcon from '@mui/icons-material/Help';
import ReportIcon from '@mui/icons-material/Report';
import Chip from '@mui/material/Chip';
import type { FinishStatus } from '../../hooks/useTaskLogs';

export type BadgeStatus = FinishStatus | 'stream-error' | 'executor-gone';

const MAP: Record<
  BadgeStatus,
  {
    label: string;
    color: 'success' | 'error' | 'warning' | 'default';
    icon: React.ReactElement;
  }
> = {
  success: { label: 'Done', color: 'success', icon: <CheckCircleIcon /> },
  stopped: { label: 'Stopped', color: 'default', icon: <CancelIcon /> },
  lost: { label: 'Lost', color: 'warning', icon: <HelpIcon /> },
  failed: { label: 'Failed', color: 'error', icon: <ReportIcon /> },
  'stream-error': {
    label: 'Stream error',
    color: 'error',
    icon: <ErrorIcon />,
  },
  'executor-gone': {
    label: 'Not in executor',
    color: 'warning',
    icon: <CloudOffIcon />,
  },
};

export interface StatusBadgeProps {
  status: BadgeStatus;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const entry = MAP[status];
  return (
    <Chip
      size="small"
      color={entry.color}
      icon={entry.icon}
      label={entry.label}
    />
  );
}
