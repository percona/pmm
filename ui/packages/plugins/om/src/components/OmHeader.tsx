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

import { Stack, Typography } from '@mui/material';

/**
 * Page title, subtitle, and a slot for page-level actions.
 *
 * Navigation between Overview and Inventory lives in PMM's sidebar, under the
 * expandable OpenManager entry — so this deliberately renders no tabs.
 */
export function OmHeader({
  title,
  subtitle,
  actions,
}: {
  title: string;
  subtitle?: React.ReactNode;
  actions?: React.ReactNode;
}) {
  return (
    <Stack
      direction="row"
      alignItems="flex-start"
      justifyContent="space-between"
      gap={2}
      flexWrap="wrap"
    >
      <Stack gap={1}>
        <Typography variant="h4" component="h1">
          {title}
        </Typography>
        {subtitle}
      </Stack>
      {actions}
    </Stack>
  );
}
