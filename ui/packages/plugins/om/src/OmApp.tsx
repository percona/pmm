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
import {
  OM_ROUTE_INVENTORY,
  OM_ROUTE_HOSTS,
  OM_ROUTE_SERVICES,
} from './constants';
import { InventoryPage } from './InventoryPage';
import { HostsPage } from './HostsPage';
import { OverviewPage } from './OverviewPage';
import { ServicesPage } from './ServicesPage';

/**
 * OM app router. The shell mounts this at ``om/*``; the cluster overview is the
 * index route, with the service, host and refresh pages beside it.
 *
 * **All four mount here now.** Discovery used to live on its own route wrapped in
 * ``SepPage``, because it read SEP's app directly and needed a bearer minted from the
 * PMM session that no other OM page did. Reading it through pmm-managed removes the
 * bearer, and with it the gate: ``SepAuthGate`` fails closed, so a SEP that was down,
 * unconfigured or refusing the exchange blanked the page entirely rather than letting
 * it render its own error.
 *
 * There is no per-cluster, per-host or per-run route: every table renders what it
 * holds and expands in place, so a detail page would only re-show rows the reader
 * already has.
 */
export const OmApp = () => {
  return (
    <Routes>
      <Route index element={<OverviewPage />} />
      <Route path={OM_ROUTE_SERVICES} element={<ServicesPage />} />
      <Route path={OM_ROUTE_HOSTS} element={<HostsPage />} />
      <Route path={OM_ROUTE_INVENTORY} element={<InventoryPage />} />
    </Routes>
  );
};
