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
import { OM_PATH } from 'lib/constants';
import { Messages } from './OmPage.messages';

const SECTIONS = [
  { label: Messages.fleet, path: OM_PATH },
  { label: Messages.nodes, path: `${OM_PATH}/nodes` },
  { label: Messages.automations, path: `${OM_PATH}/automations` },
  { label: Messages.settings, path: `${OM_PATH}/settings` },
];

/**
 * OM's sections as the app's own navigation: the sidebar lists OM once, under
 * Apps, and these tabs move between its pages. A page below a section, such as
 * the install wizard under Nodes, keeps that section's tab current.
 */
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
