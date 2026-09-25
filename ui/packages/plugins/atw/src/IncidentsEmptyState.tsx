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

import { Button, Link, Stack, Typography } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import {
  Messages,
  SUPPORT_DIAGNOSTICS_DOCS_URL,
} from './IncidentsEmptyState.messages';

export interface IncidentsEmptyStateProps {
  /** Opens the create-incident dialog. Omit for a read-only session. */
  onCreate?: () => void;
}

export function IncidentsEmptyState({ onCreate }: IncidentsEmptyStateProps) {
  return (
    <Stack
      alignItems="center"
      justifyContent="center"
      sx={{ py: 6, px: 2 }}
      data-testid="atw-incidents-empty"
    >
      <Stack
        alignItems="center"
        textAlign="center"
        gap={1}
        sx={{ maxWidth: 480, width: '100%' }}
      >
        <Typography variant="h6">{Messages.title}</Typography>
        <Typography variant="body1" color="text.secondary">
          {Messages.description}
        </Typography>

        {onCreate && (
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={onCreate}
            sx={{ mt: 3 }}
            data-testid="atw-incidents-empty-create"
          >
            {Messages.create}
          </Button>
        )}

        <Link
          href={SUPPORT_DIAGNOSTICS_DOCS_URL}
          target="_blank"
          rel="noopener noreferrer"
          sx={{ mt: 2 }}
          data-testid="atw-incidents-empty-docs"
        >
          {Messages.documentation}
        </Link>
      </Stack>
    </Stack>
  );
}
