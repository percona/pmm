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

import { StatusIcon } from '@percona/peak-ui';
import { CLUSTER_HEALTH_LABEL } from '../../../constants';
import type { OmClusterHealth } from '../../../types';
import { HEALTH_SEVERITY } from './ClusterHealthIcon.constants';

/** A cluster state's glyph. Named only when no word beside it says the state. */
export const ClusterHealthIcon = ({
  health,
  labelled = false,
}: {
  health: OmClusterHealth;
  labelled?: boolean;
}) => (
  <StatusIcon
    size="small"
    severity={HEALTH_SEVERITY[health]}
    titleAccess={labelled ? CLUSTER_HEALTH_LABEL[health] : undefined}
    data-testid={`om-cluster-health-icon-${health}`}
  />
);
