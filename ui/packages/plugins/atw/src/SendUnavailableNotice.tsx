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
