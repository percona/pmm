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
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  LinearProgress,
  List,
  ListItem,
  ListItemText,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { sendJobDetail, useAtwSendJob, useStartSendJob } from './hooks';
import type { AtwSendLogExecution } from './types';

export interface SendDialogProps {
  open: boolean;
  incidentId: string;
  /** The executions to send, already resolved to id + label. */
  executions: AtwSendLogExecution[];
  /** Prefills the case-reference field; usually the incident's own reference. */
  defaultCaseRef?: string | null;
  /** Receives whether a send was started, so the caller can drop its selection. */
  onClose: (started: boolean) => void;
}

/**
 * Collects the support-case reference, starts a diagnostics send, and follows it
 * to a terminal status.
 *
 * The send log the POST returns *is* the job, so the dialog polls that row
 * rather than a Celery result: an attempt stays readable long after a task
 * result would have expired, which is what makes "Send again" on a failed row
 * possible.
 *
 * Callers mount this under a per-open `key` so every open starts from a clean
 * mutation and poll; nothing here resets state on reopen.
 */
export function SendDialog({
  open,
  incidentId,
  executions,
  defaultCaseRef,
  onClose,
}: SendDialogProps) {
  const [caseRef, setCaseRef] = useState(defaultCaseRef ?? '');
  const [jobId, setJobId] = useState<string | null>(null);

  const startMutation = useStartSendJob(incidentId);
  const { data: job, error: pollError } = useAtwSendJob(
    incidentId,
    open ? jobId : null
  );

  const detail = sendJobDetail(job);
  const status = job?.status;
  const succeeded = status === 'success';
  const failed = status === 'failed';
  const startError = startMutation.error;
  const isSending =
    startMutation.isPending ||
    (jobId !== null && !succeeded && !failed && !pollError);

  const handleSend = () => {
    startMutation.mutate(
      {
        case_ref: caseRef.trim(),
        execution_ids: executions.map((execution) => execution.id),
      },
      { onSuccess: (created) => setJobId(created.id) }
    );
  };

  const handleClose = () => onClose(jobId !== null);

  const canSend =
    caseRef.trim().length > 0 && executions.length > 0 && !isSending;

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      fullWidth
      maxWidth="sm"
      aria-labelledby="atw-send-dialog-title"
    >
      <DialogTitle id="atw-send-dialog-title">Send to support case</DialogTitle>
      <DialogContent>
        {succeeded ? (
          <Alert severity="success" sx={{ mt: 1 }}>
            Diagnostics sent
            {detail.upload_reference
              ? ` (reference ${detail.upload_reference})`
              : ''}
            .
          </Alert>
        ) : (
          <>
            <DialogContentText sx={{ mb: 2 }}>
              The selected executions&apos; output files are bundled and
              attached to the support case.
            </DialogContentText>

            <TextField
              autoFocus
              fullWidth
              required
              size="small"
              label="Support case reference"
              value={caseRef}
              onChange={(event) => setCaseRef(event.target.value)}
              disabled={isSending}
              sx={{ mb: 2 }}
            />

            <Typography variant="subtitle2">
              {executions.length} execution{executions.length === 1 ? '' : 's'}{' '}
              selected
            </Typography>
            <List dense disablePadding>
              {executions.map((execution) => (
                <ListItem key={execution.id} disableGutters>
                  <ListItemText
                    primary={execution.snippet_filename}
                    secondary={`Task #${execution.task_history_id}`}
                  />
                </ListItem>
              ))}
            </List>

            {executions.length === 0 && (
              <Alert severity="warning" sx={{ mt: 1 }}>
                None of the selected executions still exist on this incident.
              </Alert>
            )}

            {isSending && (
              <Stack spacing={1} sx={{ mt: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  {status === 'running' ? 'Bundling and uploading…' : 'Queued…'}
                </Typography>
                <LinearProgress />
              </Stack>
            )}

            {failed && (
              <Alert severity="error" sx={{ mt: 2 }}>
                {detail.error ?? 'The send failed.'}
              </Alert>
            )}

            {startError && (
              <Alert severity="error" sx={{ mt: 2 }}>
                {startError.message}
              </Alert>
            )}

            {pollError && (
              <Alert severity="error" sx={{ mt: 2 }}>
                Lost track of this send: {pollError.message}. Check the send
                history for its outcome.
              </Alert>
            )}
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>
          {succeeded || isSending ? 'Close' : 'Cancel'}
        </Button>
        {!succeeded && (
          <Button variant="contained" onClick={handleSend} disabled={!canSend}>
            {failed || startError || pollError ? 'Send again' : 'Send'}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}
