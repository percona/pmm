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

import type { OmClusterRow } from '../../../types';
import { Messages } from '../../FleetClustersTab.messages';

const TEXT = Messages.healthReason;

export interface HealthReason {
  text: string;
  /** Every member the text sums up, when it names fewer than all of them. */
  members?: string[];
}

/** Why a cluster is in its state, in a line; nothing for a healthy one. */
export const healthReason = (
  cluster: OmClusterRow
): HealthReason | undefined => {
  const { health, services, total_services: total } = cluster;
  const down = services
    .filter((service) => service.status === 'SERVICE_STATUS_DOWN')
    .map((service) => service.service_name);
  switch (health) {
    case 'down':
      return { text: TEXT.allDown(total) };
    case 'degraded':
      return down.length === 1
        ? { text: TEXT.oneDown(down[0]) }
        : { text: TEXT.someDown(down.length, total), members: down };
    case 'unknown': {
      if (!total) {
        return { text: TEXT.noMembers };
      }
      const silent = services
        .filter((service) => service.status === 'SERVICE_STATUS_UNSPECIFIED')
        .map((service) => service.service_name);
      return {
        text: TEXT.notReporting(silent.length, total),
        members: silent,
      };
    }
    default:
      return undefined;
  }
};
