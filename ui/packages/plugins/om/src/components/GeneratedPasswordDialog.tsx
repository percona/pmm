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
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  InputAdornment,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import CheckIcon from '@mui/icons-material/Check';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
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
  const [revealed, setRevealed] = useState(false);
  const clipboard = useCopyToClipboard();
  const revealLabel = revealed ? 'Hide password' : 'Show password';
  const copyLabel = clipboard.copied ? 'Copied' : 'Copy password';

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
            type={revealed ? 'text' : 'password'}
            autoComplete="off"
            fullWidth
            helperText={
              clipboard.failed
                ? 'Copying is not available here. Show the password and copy it by hand.'
                : ' '
            }
            slotProps={{
              htmlInput: { readOnly: true, sx: { fontFamily: 'monospace' } },
              input: {
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      size="small"
                      aria-label={revealLabel}
                      title={revealLabel}
                      onClick={() => setRevealed((shown) => !shown)}
                    >
                      {revealed ? <VisibilityOffIcon /> : <VisibilityIcon />}
                    </IconButton>
                    <IconButton
                      edge="end"
                      size="small"
                      aria-label={copyLabel}
                      title={copyLabel}
                      color={clipboard.failed ? 'error' : undefined}
                      onClick={() => void clipboard.copy(password)}
                    >
                      {clipboard.copied ? (
                        <CheckIcon color="success" />
                      ) : (
                        <ContentCopyIcon />
                      )}
                    </IconButton>
                  </InputAdornment>
                ),
              },
            }}
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button variant="contained" onClick={onSaved}>
          I have saved it
        </Button>
      </DialogActions>
    </Dialog>
  );
};
