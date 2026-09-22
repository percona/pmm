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

import { Alert, Button } from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';
import { useDeliverySettingsPath } from './deliverySettings';

/**
 * Why the send controls are inert, stated once above them.
 *
 * The tooltips on each disabled control carry the backend's own reasons, but a
 * tooltip on a disabled button is not something an operator finds by accident —
 * and an administrator looking at a greyed-out Send is the one person who can
 * fix it. So the pane says it in the open and, when the host supplied a route,
 * offers the way there; the specific reason stays in the tooltips rather than
 * being repeated here, which keeps this to one line whatever the backend says.
 *
 * Rendered only for a session that has send controls to explain: a read-only
 * session is never offered one, so the connection state changes nothing it
 * could do and the notice would be noise. That also makes the settings button
 * safe to offer unconditionally — `canMutate` is the administrator flag today,
 * and the settings tab it links to is administrator-only. Should `canMutate`
 * ever widen to a lesser role, gate the button separately or it becomes the
 * same dead end this notice replaced.
 *
 * Also shown on the incident list landing page (PMM-15515) so an admin learns
 * delivery is missing before opening an incident.
 */
export function SendUnavailableNotice() {
  const settingsPath = useDeliverySettingsPath();

  return (
    <Alert
      severity="info"
      variant="outlined"
      sx={{ mb: 2, alignItems: 'center' }}
      data-testid="atw-send-unavailable"
      action={
        settingsPath ? (
          <Button
            size="small"
            component={RouterLink}
            to={settingsPath}
            data-testid="atw-send-unavailable-settings"
          >
            ServiceNow settings
          </Button>
        ) : undefined
      }
    >
      Sending requires a valid ServiceNow connection.
    </Alert>
  );
}
