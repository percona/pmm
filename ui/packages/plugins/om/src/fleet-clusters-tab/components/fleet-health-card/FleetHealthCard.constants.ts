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

import type { OmClusterHealth } from '../../../types';
import { HEALTH_FILTER_VALUES } from '../../FleetClustersTab.constants';

/** Every state but healthy, so the filter still matches a cluster that changes state later. */
export const UNHEALTHY_STATES = HEALTH_FILTER_VALUES.filter(
  (health) => health !== 'healthy'
);

/** The bar's segment colour per state, matching its status icon. */
export const HEALTH_BAR_COLOR: Record<OmClusterHealth, string> = {
  down: 'error.main',
  degraded: 'warning.main',
  unknown: 'action.disabled',
  healthy: 'success.main',
};
