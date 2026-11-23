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

import {
  OM_ROUTE_AUTOMATIONS,
  OM_ROUTE_FLEET,
  OM_ROUTE_NODES,
  OM_ROUTE_SETTINGS,
} from '@pmm/plugins-om';
import { OM_PATH } from 'lib/constants';
import { Messages } from './OmPage.messages';

export const OM_SECTIONS = [
  { label: Messages.fleet, route: OM_ROUTE_FLEET },
  { label: Messages.nodes, route: OM_ROUTE_NODES },
  { label: Messages.automations, route: OM_ROUTE_AUTOMATIONS },
  { label: Messages.settings, route: OM_ROUTE_SETTINGS },
].map(({ label, route }) => ({
  label,
  path: route ? `${OM_PATH}/${route}` : OM_PATH,
}));
