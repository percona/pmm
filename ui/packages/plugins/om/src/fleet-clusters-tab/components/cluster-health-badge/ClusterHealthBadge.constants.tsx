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
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import HelpIcon from '@mui/icons-material/Help';
import ReportIcon from '@mui/icons-material/Report';
import WarningRoundedIcon from '@mui/icons-material/WarningRounded';
import type { OmClusterHealth } from '../../../types';

/** The shapes of peak-ui's StatusIcon, at a size a small chip holds. */
export const HEALTH_CHIP_ICON: Record<OmClusterHealth, ReactElement> = {
  down: <ReportIcon />,
  degraded: <WarningRoundedIcon />,
  unknown: <HelpIcon />,
  healthy: <CheckCircleIcon />,
};
