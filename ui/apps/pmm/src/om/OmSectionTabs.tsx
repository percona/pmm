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

import Box from '@mui/material/Box';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import type { FC } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  OM_ROUTE_AUTOMATIONS,
  OM_ROUTE_FLEET,
  OM_ROUTE_NODES,
  OM_ROUTE_SETTINGS,
} from '@pmm/plugins-om';
import { OM_PATH } from 'lib/constants';
import { Messages } from './OmPage.messages';

const SECTIONS = [
  { label: Messages.fleet, route: OM_ROUTE_FLEET },
  { label: Messages.nodes, route: OM_ROUTE_NODES },
  { label: Messages.automations, route: OM_ROUTE_AUTOMATIONS },
  { label: Messages.settings, route: OM_ROUTE_SETTINGS },
].map(({ label, route }) => ({
  label,
  path: route ? `${OM_PATH}/${route}` : OM_PATH,
}));

/** Moves between OM's pages, since the sidebar lists OM once; a page below a section keeps its tab current. */
export const OmSectionTabs: FC = () => {
  const { pathname } = useLocation();
  const current =
    [...SECTIONS]
      .reverse()
      .find(({ path }) => pathname === path || pathname.startsWith(`${path}/`))
      ?.path ?? OM_PATH;

  // Tabs would put the name on its inner list, so the landmark around it
  // carries it instead.
  return (
    <Box component="nav" aria-label={Messages.sections}>
      <Tabs value={current} variant="scrollable" scrollButtons="auto">
        {SECTIONS.map(({ label, path }) => (
          <Tab
            key={path}
            value={path}
            label={label}
            component={Link}
            to={path}
            aria-current={path === current ? 'page' : undefined}
          />
        ))}
      </Tabs>
    </Box>
  );
};
