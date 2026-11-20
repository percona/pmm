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
import { OM_SECTIONS } from './OmSectionTabs.constants';

/** Moves between OM's pages, since the sidebar lists OM once; a page below a section keeps its tab current. */
export const OmSectionTabs: FC = () => {
  // Lowercased because the router matches routes regardless of case.
  const pathname = useLocation().pathname.toLowerCase();
  const current =
    [...OM_SECTIONS]
      .reverse()
      .find(({ path }) => pathname === path || pathname.startsWith(`${path}/`))
      ?.path ?? OM_PATH;

  // Tabs would put the name on its inner list, so the landmark around it
  // carries it instead.
  return (
    <Box component="nav" aria-label={Messages.sections}>
      <Tabs value={current} variant="scrollable" scrollButtons="auto">
        {OM_SECTIONS.map(({ label, path }) => (
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
