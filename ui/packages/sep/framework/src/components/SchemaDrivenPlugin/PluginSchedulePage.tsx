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
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { Route, Routes, useNavigate } from 'react-router-dom';
import type { PluginSchema } from '@sep/api';
import { ScheduledTasksPanel } from '../ScheduledTasksPanel';
import {
  resolveItemDisplayName,
  resolveItemDisplayNamePlural,
} from '../../utils/itemLabels';
import { PluginScheduleFormPage } from './PluginScheduleFormPage';

interface PluginSchedulePageProps {
  pluginName: string;
  schema: PluginSchema;
}

function ScheduleListPage({ pluginName, schema }: PluginSchedulePageProps) {
  const navigate = useNavigate();
  const itemName = resolveItemDisplayName(schema);
  const itemNamePlural = resolveItemDisplayNamePlural(schema);

  return (
    <Box>
      <Stack direction="row" alignItems="center" gap={1} sx={{ mb: 3 }}>
        {/*
          Path-relative, not route-relative: this page is the index of a splat
          route, so the number of route segments above it is not the number of
          path segments. `..` off `<plugin>/schedule` is the plugin's list.
        */}
        <IconButton
          onClick={() => navigate('..', { relative: 'path' })}
          aria-label="Back to list"
        >
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h4">Schedules</Typography>
      </Stack>

      <ScheduledTasksPanel
        pluginName={pluginName}
        displayName={schema.display_name}
        itemName={itemName}
        itemNamePlural={itemNamePlural}
        onCreate={() => navigate('new', { relative: 'path' })}
        onEdit={(task) => navigate(`${task.id}/edit`, { relative: 'path' })}
      />
    </Box>
  );
}

/**
 * The schedules area of a plugin: the list, plus the create and edit pages it
 * links to.
 *
 * Mounted under a splat route (`schedule/*`), because editing a schedule is a
 * page now rather than an expanded table row (PMM-15456).
 */
export function PluginSchedulePage({
  pluginName,
  schema,
}: PluginSchedulePageProps) {
  const itemName = resolveItemDisplayName(schema);
  const itemNamePlural = resolveItemDisplayNamePlural(schema);

  return (
    <Routes>
      <Route
        index
        element={<ScheduleListPage pluginName={pluginName} schema={schema} />}
      />
      <Route
        path="new"
        element={
          <PluginScheduleFormPage
            pluginName={pluginName}
            mode="create"
            itemName={itemName}
            itemNamePlural={itemNamePlural}
          />
        }
      />
      <Route
        path=":id/edit"
        element={
          <PluginScheduleFormPage
            pluginName={pluginName}
            mode="edit"
            itemName={itemName}
            itemNamePlural={itemNamePlural}
          />
        }
      />
    </Routes>
  );
}
