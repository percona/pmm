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
 * A router is no replica-set member, up or down. Any other service that is down
 * reports nothing this run, so its state is unobserved: `replication_set` comes from
 * the optional `--replication-set` flag, so a down member registered with only
 * `--cluster` cannot be told apart from a down standalone.
 */
export const MemberState = ({
  service,
}: {
  service: Pick<OmService, 'state' | 'status' | 'process_role'>;
}) => {
  if (service.state) {
    return <>{service.state}</>;
  }
  const unobserved =
    service.process_role !== 'PROCESS_ROLE_MONGOS' &&
    service.status === 'SERVICE_STATUS_DOWN';
  return (
    <Unavailable
      reason={unobserved ? 'service_not_observed' : 'not_applicable'}
    />
  );
};
