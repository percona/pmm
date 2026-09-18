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

import Chip from '@mui/material/Chip';
import {
  RUN_STATUS_COLOR,
  RUN_STATUS_LABEL,
  SERVICE_STATUS_COLOR,
  SERVICE_STATUS_LABEL,
} from '../constants';
import { isRunActive } from '../api';
import type { OmTopologyRunStatus, OmServiceStatus } from '../types';

/**
 * Service reachability as a chip.
 *
 * DOWN covers both "the exporter said 0" and "the service produced no metrics at
 * all" — the worker collapses them deliberately, because from the estate's point of
 * view an unreachable service and an unmonitored one are the same problem.
 */
export const StatusBadge = ({ status }: { status: OmServiceStatus }) => {
  return (
    <Chip
      size="small"
      label={SERVICE_STATUS_LABEL[status] ?? status}
      color={SERVICE_STATUS_COLOR[status] ?? 'default'}
      variant={status === 'SERVICE_STATUS_DOWN' ? 'outlined' : 'filled'}
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
