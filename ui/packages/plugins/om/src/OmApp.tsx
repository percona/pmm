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
  OM_LEGACY_REDIRECTS,
  OM_ROUTE_AUTOMATIONS,
  OM_ROUTE_INSTALL,
  OM_ROUTE_NODES,
  OM_ROUTE_SETTINGS,
} from './constants';
import { AutomationsPage } from './AutomationsPage';
import { BootstrapPage } from './BootstrapPage';
import { FleetPage } from './FleetPage';
import { LegacyRedirect } from './LegacyRedirect';
import { NodesPage } from './NodesPage';
import { SettingsPage } from './SettingsPage';
import { ScanFeedbackProvider } from './ScanFeedback';

/**
 * Operations' router. The shell mounts this at ``operations/*``; the fleet is the index
 * route, with nodes, automations and settings beside it.
 *
 * Four routes, one per job, each owning its own tabs -- so the two readings of the
 * snapshot live on Fleet, and a scan's history lives beside an install's on
 * Automations rather than on a page called Inventory, which collided with PMM's own.
 * See om-design-review/structure-and-glossary-proposal.md for why that is the rule.
 *
 * There is no per-cluster or per-run route: every table renders what it holds and
 * expands in place, so a detail page would only re-show rows the reader already has.
 * The install wizard is the one exception -- it needs the node selection `NodesPage`
 * made, carried across as a `?nodes=` query param rather than duplicated as a second
 * selection UI (see {@link BootstrapPage}'s own doc comment).
 *
 * The legacy redirects exist because the feature build is already out with people
 * clicking round it; see {@link OM_LEGACY_REDIRECTS}.
 */
export const OmApp = () => {
  return (
    <ScanFeedbackProvider>
      <Routes>
        <Route index element={<FleetPage />} />
        <Route path={OM_ROUTE_NODES} element={<NodesPage />} />
        <Route path={OM_ROUTE_INSTALL} element={<BootstrapPage />} />
        <Route path={OM_ROUTE_AUTOMATIONS} element={<AutomationsPage />} />
        <Route path={OM_ROUTE_SETTINGS} element={<SettingsPage />} />
        {Object.entries(OM_LEGACY_REDIRECTS).map(([from, to]) => (
          <Route key={from} path={from} element={<LegacyRedirect to={to} />} />
        ))}
      </Routes>
    </ScanFeedbackProvider>
  );
};
