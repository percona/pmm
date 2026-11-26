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
import { CLUSTER_HEALTH_COLOR, CLUSTER_HEALTH_LABEL } from '../../../constants';
import type { OmClusterHealth } from '../../../types';
import { ClusterHealthIcon } from '../cluster-health-icon';

/** A cluster's state as an outlined chip: the tab's status icon, word and color. */
export const ClusterHealthBadge = ({ health }: { health: OmClusterHealth }) => (
  <Chip
    size="small"
    variant="outlined"
    icon={<ClusterHealthIcon health={health} />}
    label={CLUSTER_HEALTH_LABEL[health]}
    color={CLUSTER_HEALTH_COLOR[health]}
    data-testid="om-cluster-health"
  />
);
