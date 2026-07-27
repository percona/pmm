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

import { useState } from 'react';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  IconButton,
  Link as MuiLink,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import { Link } from 'react-router-dom';
import {
  useAtwIncidents,
  useCreateAtwIncident,
  useDeleteAtwIncident,
  useUpdateAtwIncident,
} from './hooks';
import type { AtwIncident } from './types';

/**
 * Landing page rendered at ``/atw``: the incident list. Supports creating an
 * incident (optional name — the server defaults it to a timestamp), renaming,
 * deleting, and opening one into its workspace.
 */
export function IncidentListPage() {
  const { data: incidents, isLoading, error } = useAtwIncidents();
  const createMutation = useCreateAtwIncident();
  const updateMutation = useUpdateAtwIncident();
  const deleteMutation = useDeleteAtwIncident();

  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState('');
  const [renameTarget, setRenameTarget] = useState<AtwIncident | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<AtwIncident | null>(null);

  const handleCreate = () => {
    const name = createName.trim();
    createMutation.mutate(name ? { name } : {}, {
      onSuccess: () => {
        setCreateOpen(false);
        setCreateName('');
      },
    });
  };

  const handleRename = () => {
    if (!renameTarget) {
      return;
    }
    const name = renameValue.trim();
    if (!name) {
      return;
    }
    updateMutation.mutate(
      { incidentId: renameTarget.id, body: { name } },
      { onSuccess: () => setRenameTarget(null) }
    );
  };

  const handleDelete = () => {
    if (!deleteTarget) {
      return;
    }
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => setDeleteTarget(null),
    });
  };

  return (
    <Box>
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        sx={{ mb: 1 }}
      >
        <Typography variant="h4">Collect Diagnostic Data</Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => {
            createMutation.reset();
            setCreateName('');
            setCreateOpen(true);
          }}
        >
          New incident
        </Button>
      </Stack>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        Open an incident to run diagnostic snippets and review their results in
        one place.
      </Typography>

      {isLoading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
          <CircularProgress />
        </Box>
      )}

      {error && (
        <Alert severity="error">
          Failed to load incidents: {error.message}
        </Alert>
      )}

      {!isLoading && !error && (!incidents || incidents.length === 0) && (
        <Alert severity="info">
          No incidents yet. Create one to get started.
        </Alert>
      )}

      {incidents && incidents.length > 0 && (
        <Stack spacing={1}>
          {incidents.map((incident) => (
            <Box
              key={incident.id}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1,
                p: 1.5,
                border: 1,
                borderColor: 'divider',
                borderRadius: 1,
              }}
            >
              <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                <MuiLink component={Link} to={incident.id} variant="subtitle1">
                  {incident.name}
                </MuiLink>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  display="block"
                >
                  {incident.case_ref ? `Case ${incident.case_ref} · ` : ''}
                  Created by {incident.created_by}
                </Typography>
              </Box>
              <Tooltip title="Rename">
                <IconButton
                  aria-label={`Rename ${incident.name}`}
                  onClick={() => {
                    updateMutation.reset();
                    setRenameTarget(incident);
                    setRenameValue(incident.name);
                  }}
                >
                  <EditOutlinedIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Tooltip title="Delete">
                <IconButton
                  aria-label={`Delete ${incident.name}`}
                  onClick={() => {
                    deleteMutation.reset();
                    setDeleteTarget(incident);
                  }}
                >
                  <DeleteOutlineIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Box>
          ))}
        </Stack>
      )}

      {/* Create */}
      <Dialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        fullWidth
        maxWidth="sm"
        aria-labelledby="atw-create-incident-title"
      >
        <DialogTitle id="atw-create-incident-title">New incident</DialogTitle>
        <DialogContent>
          {createMutation.isError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {createMutation.error?.message ?? 'Failed to create incident'}
            </Alert>
          )}
          <TextField
            autoFocus
            fullWidth
            margin="dense"
            label="Name (optional)"
            helperText="Leave blank to use a timestamped default."
            value={createName}
            onChange={(event) => setCreateName(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault();
                handleCreate();
              }
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleCreate}
            loading={createMutation.isPending}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>

      {/* Rename */}
      <Dialog
        open={renameTarget !== null}
        onClose={() => setRenameTarget(null)}
        fullWidth
        maxWidth="sm"
        aria-labelledby="atw-rename-incident-title"
      >
        <DialogTitle id="atw-rename-incident-title">
          Rename incident
        </DialogTitle>
        <DialogContent>
          {updateMutation.isError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {updateMutation.error?.message ?? 'Failed to rename incident'}
            </Alert>
          )}
          <TextField
            autoFocus
            fullWidth
            margin="dense"
            label="Name"
            value={renameValue}
            onChange={(event) => setRenameValue(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault();
                handleRename();
              }
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRenameTarget(null)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleRename}
            loading={updateMutation.isPending}
            disabled={renameValue.trim().length === 0}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete */}
      <Dialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        aria-labelledby="atw-delete-incident-title"
      >
        <DialogTitle id="atw-delete-incident-title">
          Delete incident
        </DialogTitle>
        <DialogContent>
          {deleteMutation.isError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {deleteMutation.error?.message ?? 'Failed to delete incident'}
            </Alert>
          )}
          <DialogContentText>
            Delete “{deleteTarget?.name}”? Its recorded executions are removed.
            This cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button
            color="error"
            variant="contained"
            onClick={handleDelete}
            loading={deleteMutation.isPending}
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
