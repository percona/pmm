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

import {
  Alert,
  Box,
  CircularProgress,
  Link as MuiLink,
  Paper,
  Typography,
} from '@mui/material';
import { useNavigate, useParams } from 'react-router-dom';
import { CollectPane } from './CollectPane';
import { ResultsPane } from './ResultsPane';
import { useAtwIncident } from './hooks';

/**
 * Incident workspace rendered at ``/atw/:incidentId``. Two side-by-side panes —
 * Collect (browse, select, and batch-execute snippets) and Results (each
 * execution's status, logs, and file listing) — stacked on narrow screens.
 */
export function IncidentWorkspacePage() {
  const { incidentId } = useParams<{ incidentId: string }>();
  const navigate = useNavigate();
  const { data: incident, isLoading, error } = useAtwIncident(incidentId);

  if (!incidentId) {
    return null;
  }

  return (
    <Box>
      <MuiLink
        component="button"
        type="button"
        onClick={() => navigate('..')}
        sx={{ mb: 2, display: 'inline-block' }}
      >
        ← Back to incidents
      </MuiLink>

      {isLoading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      )}

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          Failed to load incident: {error.message}
        </Alert>
      )}

      <Typography variant="h4" sx={{ mb: 0.5 }}>
        {incident?.name ?? 'Incident'}
      </Typography>
      {incident?.case_ref && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          Case reference: {incident.case_ref}
        </Typography>
      )}

      <Box
        sx={{
          mt: 1,
          display: 'grid',
          gap: 2,
          gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
          alignItems: 'start',
        }}
      >
        <Paper variant="outlined" sx={{ p: 2 }}>
          <CollectPane incidentId={incidentId} />
        </Paper>
        <Paper variant="outlined" sx={{ p: 2 }}>
          <ResultsPane incidentId={incidentId} />
        </Paper>
      </Box>
    </Box>
  );
}
