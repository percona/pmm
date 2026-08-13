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
import { OverviewPage } from './OverviewPage';
import { TopologyPage } from './TopologyPage';

/**
 * POM app router. The shell mounts this at ``pom/*``; the cluster overview is the
 * index route and the full service table lives at ``topology``.
 *
 * ``runs`` is deliberately absent. Discovery reads SEP's ``pom_discovery`` app
 * directly, so it needs a SEP bearer that the rest of POM does not: the shell mounts
 * ``RunsPage`` on its own route inside ``SepPage``, which ranks above this splat.
 * Mounting it here as well would put an ungated copy on the same path.
 *
 * There is no per-cluster route: the snapshot is one document and the topology table
 * renders all of it, so a detail page would only re-show rows the reader already has.
 */
export function PomApp() {
  return (
    <Routes>
      <Route index element={<OverviewPage />} />
      <Route path="topology" element={<TopologyPage />} />
    </Routes>
  );
}
