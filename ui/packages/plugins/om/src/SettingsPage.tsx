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

import { Box, Typography } from '@mui/material';
import { OmHeader } from './components/OmHeader';
import { ConfigForm } from './components/ConfigForm';

/**
 * This app's own configuration, on its own page.
 *
 * It used to be a tab on a page called Inventory, which is two problems in one: the
 * page collided with PMM's own Inventory, and a reader looking for settings had no
 * reason to open a page about refresh history to find them. Settings is a job, so by
 * the structure's own rule it is an entry.
 *
 * Not to be confused with PMM's switch for the whole feature, which stays in PMM's
 * Settings -> Advanced. That one decides whether Operations exists; this one decides
 * how it behaves.
 */
export const SettingsPage = () => {
  return (
    <Box>
      <OmHeader
        title="Settings"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            How Operations scans your nodes, and what it collects from them.
          </Typography>
        }
      />
      <ConfigForm />
    </Box>
  );
};
