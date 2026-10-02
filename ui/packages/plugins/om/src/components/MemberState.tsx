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

import { Unavailable } from './Unavailable';
import type { OmService } from '../types';

/**
 * A service's replica-set member state, as MongoDB names it (PRIMARY, SECONDARY...).
 *
 * No state has two causes that need different words. A router or a standalone is no
 * replica-set member at all, up or down; a member that is down reports nothing this
 * run, so its state is unobserved.
 */
export const MemberState = ({
  service,
}: {
  service: Pick<
    OmService,
    'state' | 'status' | 'process_role' | 'replication_set'
  >;
}) => {
  if (service.state) {
    return <>{service.state}</>;
  }
  const isMember =
    service.process_role !== 'PROCESS_ROLE_MONGOS' && !!service.replication_set;
  return (
    <Unavailable
      reason={
        isMember && service.status === 'SERVICE_STATUS_DOWN'
          ? 'service_not_observed'
          : 'not_applicable'
      }
    />
  );
};
