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

import type { ReactNode } from 'react';
import { Box, Button, Stack, Typography } from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';

/**
 * The one empty state, for every page that can have nothing on it.
 *
 * Always in the same order: what the page shows, why it is empty, and the action that
 * changes that. The action is optional - not every page has one, and no action beats a
 * button that leads nowhere useful.
 *
 * Deliberately low emphasis: an empty page on a fresh install is expected, not a
 * problem, so it should not look like an alert.
 */
export const EmptyState = ({
  title,
  children,
  action,
}: {
  /** What this page shows, in a few words. */
  title: string;
  /** Why it is empty right now, and what would fill it. */
  children: ReactNode;
  /** The one thing to do about it, where there is one. */
  action?: { label: string; to: string };
}) => (
  <Box
    sx={{
      py: 6,
      px: 3,
      textAlign: 'center',
      border: 1,
      borderColor: 'divider',
      borderRadius: 1,
    }}
    data-testid="om-empty-state"
  >
    <Stack gap={1} alignItems="center">
      <Typography variant="subtitle1">{title}</Typography>
      <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 520 }}>
        {children}
      </Typography>
      {action && (
        <Button
          component={RouterLink}
          to={action.to}
          variant="outlined"
          size="small"
          sx={{ mt: 1 }}
        >
          {action.label}
        </Button>
      )}
    </Stack>
  </Box>
);
