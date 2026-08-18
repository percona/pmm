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

import { Route, Routes } from 'react-router-dom';
import { POM_ROUTE_HOSTS, POM_ROUTE_SERVICES } from './constants';
import { HostsPage } from './HostsPage';
import { OverviewPage } from './OverviewPage';
import { ServicesPage } from './ServicesPage';

/**
 * POM app router. The shell mounts this at ``pom/*``; the cluster overview is the
 * index route, with the service and host tables beside it.
 *
 * ``discovery`` is still deliberately absent. That page reads SEP's ``pom_discovery``
 * app directly, so it needs a SEP bearer the rest of POM does not: the shell mounts
 * ``RunsPage`` on its own route inside ``SepPage``, which ranks above this splat, and
 * mounting it here as well would put an ungated copy on the same path. It joins these
 * routes once it moves onto ``/v1/pom/inventory`` and stops needing the bearer.
 *
 * There is no per-cluster or per-host route: both tables render everything they hold,
 * and the Hosts rows expand to their services in place, so a detail page would only
 * re-show rows the reader already has.
 */
export function PomApp() {
  return (
    <Routes>
      <Route index element={<OverviewPage />} />
      <Route path={POM_ROUTE_SERVICES} element={<ServicesPage />} />
      <Route path={POM_ROUTE_HOSTS} element={<HostsPage />} />
    </Routes>
  );
}
