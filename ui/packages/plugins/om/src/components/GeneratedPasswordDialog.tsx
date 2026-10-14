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
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { useCopyToClipboard } from '@pmm-extensions/framework';

interface GeneratedPasswordDialogProps {
  username: string;
  password: string;
  onSaved: () => void;
}

/** Shows the administrator password PMM generated, the one time it is available. */
export const GeneratedPasswordDialog = ({
  username,
  password,
  onSaved,
}: GeneratedPasswordDialogProps) => {
  const clipboard = useCopyToClipboard();

  return (
    <Dialog open maxWidth="sm" fullWidth>
      <DialogTitle>Save the administrator password</DialogTitle>
      <DialogContent>
        <Stack spacing={2}>
          <Typography variant="body2">
            PMM deletes its copy once the install has created the user, so this
            is the only time it is shown. Store it somewhere safe before you
            continue.
          </Typography>
          <TextField
            label="Username"
            value={username}
            fullWidth
            slotProps={{ htmlInput: { readOnly: true } }}
          />
          <TextField
            label="Password"
            value={password}
            fullWidth
            slotProps={{
              htmlInput: { readOnly: true, sx: { fontFamily: 'monospace' } },
            }}
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button
          color={clipboard.failed ? 'error' : undefined}
          onClick={() => void clipboard.copy(password)}
        >
          {clipboard.copied
            ? 'Copied'
            : clipboard.failed
              ? 'Could not copy - select it instead'
              : 'Copy password'}
        </Button>
        <Button variant="contained" onClick={onSaved}>
          I have saved it
        </Button>
      </DialogActions>
    </Dialog>
  );
};
