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
import { IncidentListPage } from './IncidentListPage';
import { IncidentWorkspacePage } from './IncidentWorkspacePage';

/**
 * ATW app router. The shell mounts this at ``atw/*``; the incident list is the
 * index route and an incident opens its workspace at ``:incidentId``.
 */
export function AtwApp() {
  return (
    <Routes>
      <Route index element={<IncidentListPage />} />
      <Route path=":incidentId" element={<IncidentWorkspacePage />} />
    </Routes>
  );
}
