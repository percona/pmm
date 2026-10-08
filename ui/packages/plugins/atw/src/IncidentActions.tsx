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

import { useEffect, useId, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  TextField,
} from '@mui/material';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import LockOpenOutlinedIcon from '@mui/icons-material/LockOpenOutlined';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import {
  type useAtwIncidentLifecycle,
  useDeleteAtwIncident,
  useUpdateAtwIncident,
} from './hooks';
import type { AtwIncident } from './types';

type IncidentLifecycle = ReturnType<typeof useAtwIncidentLifecycle>;

/** Open or closed, read from the served `closed_at`. */
export function isIncidentClosed(incident: Pick<AtwIncident, 'closed_at'>) {
  return Boolean(incident.closed_at);
}

/**
 * The incident's state, shown beside its name in both states. Open is the
 * coloured one because it is the state that may still need someone; closed is
 * the resting state and reads as such.
 */
export function IncidentStatusChip({ incident }: { incident: AtwIncident }) {
  const closed = isIncidentClosed(incident);
  return (
    <Chip
      label={closed ? 'Closed' : 'Open'}
      size="small"
      color={closed ? 'default' : 'success'}
      variant="outlined"
      data-testid="atw-incident-status"
    />
  );
}

export interface IncidentActionsMenuProps {
  incident: AtwIncident;
  lifecycle: IncidentLifecycle;
  onRename: (incident: AtwIncident) => void;
  onDelete: (incident: AtwIncident) => void;
  /** `small` in a table row; the header uses the default size. */
  size?: 'small' | 'medium';
}

/**
 * Rename, Close/Reopen and Delete behind one labelled button — the same menu on
 * the list and in the workspace, so neither surface promotes the last thing
 * anyone does with an incident to its most prominent control, and no
 * destructive action hides behind an unlabelled icon.
 *
 * Rename and Delete are handed back to the caller: the list renames in a
 * dialog, the workspace in place, and only the workspace has to leave the page
 * after a delete.
 */
export function IncidentActionsMenu({
  incident,
  lifecycle,
  onRename,
  onDelete,
  size = 'medium',
}: IncidentActionsMenuProps) {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const menuId = useId();
  const closed = isIncidentClosed(incident);
  const pending = lifecycle.isPending(incident.id);

  // Run once the menu has finished closing, not on click: closing hands focus
  // back to the Actions button, and an action that moves focus itself — the
  // workspace's in-place rename focuses its field — would lose it to that
  // restore, blurring the field closed the moment it opened.
  const pendingActionRef = useRef<(() => void) | null>(null);
  const choose = (action: () => void) => {
    pendingActionRef.current = action;
    setAnchor(null);
  };
  const runPendingAction = () => {
    const action = pendingActionRef.current;
    pendingActionRef.current = null;
    action?.();
  };

  return (
    <>
      <Button
        variant="outlined"
        size={size}
        endIcon={<KeyboardArrowDownIcon />}
        aria-haspopup="menu"
        aria-expanded={anchor ? 'true' : undefined}
        aria-controls={anchor ? menuId : undefined}
        aria-label={`Actions for ${incident.name}`}
        onClick={(event) => {
          // Rows open their incident on click; the menu is not a way in.
          event.stopPropagation();
          setAnchor(event.currentTarget);
        }}
      >
        Actions
      </Button>
      <Menu
        id={menuId}
        anchorEl={anchor}
        open={anchor !== null}
        onClose={() => setAnchor(null)}
        onClick={(event) => event.stopPropagation()}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        slotProps={{ transition: { onExited: runPendingAction } }}
      >
        <MenuItem onClick={() => choose(() => onRename(incident))}>
          <ListItemIcon>
            <EditOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>Rename</ListItemText>
        </MenuItem>
        {closed ? (
          <MenuItem
            disabled={pending}
            onClick={() => choose(() => lifecycle.reopen(incident.id))}
          >
            <ListItemIcon>
              <LockOpenOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>Reopen</ListItemText>
          </MenuItem>
        ) : (
          <MenuItem
            disabled={pending}
            onClick={() => choose(() => lifecycle.close(incident.id))}
          >
            <ListItemIcon>
              <LockOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>Close</ListItemText>
          </MenuItem>
        )}
        <MenuItem
          onClick={() => choose(() => onDelete(incident))}
          sx={{ color: 'error.main' }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <DeleteOutlineIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>Delete</ListItemText>
        </MenuItem>
      </Menu>
    </>
  );
}

export interface RenameIncidentDialogProps {
  incident: AtwIncident | null;
  onClose: () => void;
}

/** Rename an incident from the list, where there is no title to edit in place. */
export function RenameIncidentDialog({
  incident,
  onClose,
}: RenameIncidentDialogProps) {
  const updateMutation = useUpdateAtwIncident();
  const [value, setValue] = useState('');

  useEffect(() => {
    if (incident) {
      updateMutation.reset();
      setValue(incident.name);
    }
    // Re-seeded per target only; the mutation object changes every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [incident]);

  const handleSave = () => {
    const name = value.trim();
    if (!incident || !name) {
      return;
    }
    updateMutation.mutate(
      { incidentId: incident.id, body: { name } },
      { onSuccess: onClose }
    );
  };

  return (
    <Dialog
      open={incident !== null}
      onClose={onClose}
      fullWidth
      maxWidth="sm"
      aria-labelledby="atw-rename-incident-title"
    >
      <DialogTitle id="atw-rename-incident-title">Rename incident</DialogTitle>
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
          value={value}
          onChange={(event) => setValue(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault();
              handleSave();
            }
          }}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleSave}
          loading={updateMutation.isPending}
          disabled={value.trim().length === 0}
        >
          Save
        </Button>
      </DialogActions>
    </Dialog>
  );
}

export interface DeleteIncidentDialogProps {
  incident: AtwIncident | null;
  onClose: () => void;
  /** Runs after the server confirms the delete, before the dialog closes. */
  onDeleted?: (incident: AtwIncident) => void;
}

export function DeleteIncidentDialog({
  incident,
  onClose,
  onDeleted,
}: DeleteIncidentDialogProps) {
  const deleteMutation = useDeleteAtwIncident();

  useEffect(() => {
    if (incident) {
      deleteMutation.reset();
    }
    // Reset per target only; the mutation object changes every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [incident]);

  const handleDelete = () => {
    if (!incident) {
      return;
    }
    deleteMutation.mutate(incident.id, {
      onSuccess: () => {
        onDeleted?.(incident);
        onClose();
      },
    });
  };

  return (
    <Dialog
      open={incident !== null}
      onClose={onClose}
      aria-labelledby="atw-delete-incident-title"
    >
      <DialogTitle id="atw-delete-incident-title">Delete incident</DialogTitle>
      <DialogContent>
        {deleteMutation.isError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {deleteMutation.error?.message ?? 'Failed to delete incident'}
          </Alert>
        )}
        <DialogContentText>
          Delete “{incident?.name}”? Its recorded executions are removed. This
          cannot be undone.
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
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
  );
}
