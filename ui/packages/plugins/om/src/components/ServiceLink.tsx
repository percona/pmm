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

import Link from '@mui/material/Link';
import { Link as RouterLink } from 'react-router-dom';

/**
 * PMM's MongoDB instance dashboard, inside the shell's Grafana route.
 *
 * Absolute under the router basename, like `useOmBase`'s links: the shell mounts
 * Grafana at `/graph/*`, so this resolves to `/pmm-ui/graph/d/...` from any OM page.
 */
const MONGODB_INSTANCE_DASHBOARD =
  '/graph/d/mongodb-instance-summary/mongodb-instance-summary';

/** The instance dashboard, filtered to one service by the name PMM knows it by. */
export function serviceDashboardPath(serviceName: string): string {
  const params = new URLSearchParams({ 'var-service_name': serviceName });
  return `${MONGODB_INSTANCE_DASHBOARD}?${params.toString()}`;
}

/**
 * A service's name, linking to its PMM dashboard.
 *
 * Linked whatever its status: a down member is the one a reader most wants to open,
 * and its dashboard is where the outage shows.
 */
export const ServiceLink = ({ serviceName }: { serviceName: string }) => (
  <Link
    component={RouterLink}
    to={serviceDashboardPath(serviceName)}
    underline="hover"
  >
    {serviceName}
  </Link>
);
