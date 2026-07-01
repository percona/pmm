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
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Checkbox,
  CircularProgress,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Link,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import AutorenewIcon from '@mui/icons-material/Autorenew';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import DownloadIcon from '@mui/icons-material/Download';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import { ApiError } from '@sep/api';
import {
  useSnippets,
  useApproveSnippet,
  useRemoveSnippetApproval,
  useBatchApproveSnippets,
  useSnippetsCapabilities,
  useRefreshSnippets,
  type BatchApproveError,
} from './hooks';
import type {
  BatchApprovalErrorResponse,
  BatchApprovalResponse,
  RefreshResponse,
  SnippetResponse,
} from './types';

interface SnippetsListPageProps {
  isAdmin?: boolean;
}

interface BatchResult {
  success?: BatchApprovalResponse;
  error?: BatchApprovalErrorResponse;
  generic?: string;
}

function buildSnippetDownloadUrl(filename: string) {
  return `/static/snippets/${filename.split('/').map(encodeURIComponent).join('/')}`;
}

function ApproveButton({
  snippet,
  hasDownloaded,
}: {
  snippet: SnippetResponse;
  hasDownloaded: boolean;
}) {
  const approve = useApproveSnippet(snippet.filename);
  const removeApproval = useRemoveSnippetApproval(snippet.filename);
  const isPending = approve.isPending || removeApproval.isPending;
  const spinner = <CircularProgress size={16} color="inherit" />;
  const [confirmOpen, setConfirmOpen] = useState(false);

  if (snippet.is_approved) {
    return (
      <Tooltip title="Remove approval">
        <Button
          size="small"
          color="warning"
          startIcon={isPending ? spinner : <RemoveCircleOutlineIcon />}
          disabled={isPending}
          onClick={(e) => {
            e.stopPropagation();
            removeApproval.mutate();
          }}
        >
          Remove
        </Button>
      </Tooltip>
    );
  }

  return (
    <>
      <Tooltip title="Approve snippet">
        <Button
          size="small"
          color="success"
          startIcon={isPending ? spinner : <CheckCircleOutlineIcon />}
          disabled={isPending || !hasDownloaded}
          onClick={(e) => {
            e.stopPropagation();
            setConfirmOpen(true);
          }}
        >
          Approve
        </Button>
      </Tooltip>
      <Dialog open={confirmOpen} onClose={() => setConfirmOpen(false)}>
        <DialogTitle>Approve snippet?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            You are about to approve <strong>{snippet.filename}</strong> without
            downloading and inspecting it. Only approve snippets you have
            reviewed.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmOpen(false)}>Cancel</Button>
          <Button
            color="success"
            onClick={() => {
              setConfirmOpen(false);
              approve.mutate();
            }}
          >
            Approve
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

/**
 * Snippet-centric list page rendered at `/snippets/`.
 *
 * Renders the snippet entities discovered by the backend (one row per
 * snippet file). When `isAdmin` is true, per-row approve / remove-approval
 * buttons and a multi-select batch-approve action are shown.
 */
export function SnippetsListPage({ isAdmin = false }: SnippetsListPageProps) {
  const navigate = useNavigate();
  const { data: snippets = [], isLoading, error } = useSnippets();
  const batchApprove = useBatchApproveSnippets();
  const { data: capabilities } = useSnippetsCapabilities();
  const refresh = useRefreshSnippets();

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [downloaded, setDownloaded] = useState<Set<string>>(new Set());
  const [batchResult, setBatchResult] = useState<BatchResult | null>(null);
  const [batchConfirmOpen, setBatchConfirmOpen] = useState(false);
  const [refreshConfirmOpen, setRefreshConfirmOpen] = useState(false);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [refreshSuccess, setRefreshSuccess] = useState<RefreshResponse | null>(
    null
  );

  const showRefresh = isAdmin && capabilities?.manual_sync_enabled;

  const selectableSnippets = snippets.filter((snippet) => !snippet.is_approved);
  const selectedFilenames = selectableSnippets
    .filter((snippet) => selected.has(snippet.filename))
    .map((snippet) => snippet.filename);

  function markDownloaded(filename: string) {
    setDownloaded((prev) => new Set(prev).add(filename));
  }

  function toggleSelect(snippet: SnippetResponse) {
    if (snippet.is_approved) {
      return;
    }
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(snippet.filename)) {
        next.delete(snippet.filename);
      } else {
        next.add(snippet.filename);
      }
      return next;
    });
  }

  function toggleSelectAll() {
    const selectableFilenames = selectableSnippets.map((s) => s.filename);
    const allSelectableSelected =
      selectableFilenames.length > 0 &&
      selectableFilenames.every((filename) => selected.has(filename));
    if (allSelectableSelected) {
      setSelected(new Set());
    } else {
      setSelected(new Set(selectableFilenames));
    }
  }

  function handleBatchApprove() {
    setBatchResult(null);
    setBatchConfirmOpen(false);
    batchApprove.mutate(
      { filenames: selectedFilenames },
      {
        onSuccess: (data) => {
          setBatchResult({ success: data });
          setSelected(new Set());
        },
        onError: (err: BatchApproveError) => {
          if (err.structured) {
            setBatchResult({ error: err.detail });
          } else {
            setBatchResult({
              generic: 'Batch approval failed. Please try again.',
            });
          }
        },
      }
    );
  }

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ py: 4 }}>
        <Alert severity="error">Failed to load snippets: {error.message}</Alert>
      </Box>
    );
  }

  const selectableCount = selectableSnippets.length;
  const selectedSelectableCount = selectedFilenames.length;
  const allSelected =
    selectableCount > 0 && selectedSelectableCount === selectableCount;
  const someSelected =
    selectedSelectableCount > 0 && selectedSelectableCount < selectableCount;

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 1 }}>
        Snippet Manager
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        Pre-approved snippets discovered from disk. Click a row to view its
        execution form and history.
      </Typography>

      {isAdmin && batchResult?.success && (
        <Alert
          severity="success"
          onClose={() => setBatchResult(null)}
          sx={{ mb: 2 }}
        >
          <Stack direction="row" spacing={1} flexWrap="wrap">
            <Chip
              label={`${batchResult.success.count} approved`}
              color="success"
              size="small"
            />
            {batchResult.success.skipped_already_approved.length > 0 && (
              <Chip
                label={`${batchResult.success.skipped_already_approved.length} already approved`}
                color="default"
                size="small"
              />
            )}
          </Stack>
        </Alert>
      )}

      {isAdmin && batchResult?.generic && (
        <Alert
          severity="error"
          onClose={() => setBatchResult(null)}
          sx={{ mb: 2 }}
        >
          {batchResult.generic}
        </Alert>
      )}

      {isAdmin && batchResult?.error && (
        <Alert
          severity="error"
          onClose={() => setBatchResult(null)}
          sx={{ mb: 2 }}
        >
          <Stack spacing={0.5}>
            {batchResult.error.missing_in_db.length > 0 && (
              <Box>
                <strong>Missing in database:</strong>{' '}
                {batchResult.error.missing_in_db.map((f) => (
                  <Chip key={f} label={f} size="small" sx={{ mr: 0.5 }} />
                ))}
              </Box>
            )}
            {batchResult.error.missing_on_disk.length > 0 && (
              <Box>
                <strong>Missing on disk:</strong>{' '}
                {batchResult.error.missing_on_disk.map((f) => (
                  <Chip key={f} label={f} size="small" sx={{ mr: 0.5 }} />
                ))}
              </Box>
            )}
          </Stack>
        </Alert>
      )}

      {isAdmin && selectedFilenames.length > 0 && (
        <Box sx={{ mb: 2 }}>
          <Button
            variant="contained"
            color="success"
            onClick={() => setBatchConfirmOpen(true)}
            disabled={batchApprove.isPending}
          >
            Batch approve ({selectedFilenames.length})
          </Button>
        </Box>
      )}

      <Dialog
        open={batchConfirmOpen}
        onClose={() => setBatchConfirmOpen(false)}
      >
        <DialogTitle>Approve selected snippets?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Approve selected snippets? They will be approved WITHOUT being
            downloaded first — the per-row download-before-approve guard is
            bypassed for batch approval.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setBatchConfirmOpen(false)}>Cancel</Button>
          <Button
            color="success"
            onClick={handleBatchApprove}
            disabled={batchApprove.isPending}
          >
            Approve selected
          </Button>
        </DialogActions>
      </Dialog>

      {showRefresh && (
        <Box sx={{ mb: 2 }}>
          <Button
            variant="outlined"
            startIcon={
              refresh.isPending ? (
                <CircularProgress size={16} color="inherit" />
              ) : (
                <AutorenewIcon />
              )
            }
            disabled={refresh.isPending}
            onClick={() => setRefreshConfirmOpen(true)}
          >
            Refresh snippets
          </Button>
        </Box>
      )}

      {refreshSuccess && (
        <Alert
          severity="success"
          onClose={() => setRefreshSuccess(null)}
          sx={{ mb: 2 }}
        >
          Snippets refreshed at{' '}
          {new Date(refreshSuccess.refreshed_at).toLocaleString()}.
        </Alert>
      )}

      {refreshError && (
        <Alert
          severity="error"
          onClose={() => setRefreshError(null)}
          sx={{ mb: 2 }}
        >
          {refreshError}
        </Alert>
      )}

      <Dialog
        open={refreshConfirmOpen}
        onClose={() => setRefreshConfirmOpen(false)}
      >
        <DialogTitle>Refresh snippets?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to refresh the saved snippets now?
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRefreshConfirmOpen(false)}>Cancel</Button>
          <Button
            color="primary"
            disabled={refresh.isPending}
            onClick={() => {
              setRefreshConfirmOpen(false);
              setRefreshError(null);
              setRefreshSuccess(null);
              refresh.mutate(undefined, {
                onSuccess: (data) => {
                  setRefreshSuccess(data);
                  setDownloaded(new Set());
                  setSelected(new Set());
                },
                onError: (err) => {
                  const detail =
                    err instanceof ApiError &&
                    err.kind === 'http' &&
                    err.message
                      ? err.message
                      : 'Snippet refresh failed. Please try again.';
                  setRefreshError(detail);
                },
              });
            }}
          >
            Refresh
          </Button>
        </DialogActions>
      </Dialog>

      {snippets.length === 0 ? (
        <Typography color="text.secondary">No snippets available.</Typography>
      ) : (
        <Table size="small">
          <TableHead>
            <TableRow>
              {isAdmin && (
                <TableCell padding="checkbox">
                  <Checkbox
                    indeterminate={someSelected}
                    checked={allSelected}
                    disabled={selectableCount === 0}
                    onChange={toggleSelectAll}
                    inputProps={{ 'aria-label': 'select all snippets' }}
                  />
                </TableCell>
              )}
              <TableCell>Filename</TableCell>
              <TableCell>Title</TableCell>
              <TableCell>Description</TableCell>
              <TableCell>Approved</TableCell>
              <TableCell>Reason</TableCell>
              {isAdmin && <TableCell>Actions</TableCell>}
            </TableRow>
          </TableHead>
          <TableBody>
            {snippets.map((snippet) => (
              <TableRow
                key={snippet.filename}
                hover
                sx={{ cursor: 'pointer' }}
                onClick={() => navigate(encodeURIComponent(snippet.filename))}
              >
                {isAdmin && (
                  <TableCell
                    padding="checkbox"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Checkbox
                      checked={selected.has(snippet.filename)}
                      disabled={snippet.is_approved}
                      onChange={() => toggleSelect(snippet)}
                      inputProps={{
                        'aria-label': `select ${snippet.filename}`,
                      }}
                    />
                  </TableCell>
                )}
                <TableCell>
                  <Link
                    component="button"
                    type="button"
                    underline="hover"
                    onClick={(event) => {
                      event.stopPropagation();
                      navigate(encodeURIComponent(snippet.filename));
                    }}
                  >
                    {snippet.filename}
                  </Link>
                </TableCell>
                <TableCell>{snippet.title}</TableCell>
                <TableCell>{snippet.description}</TableCell>
                <TableCell>{snippet.is_approved ? 'Yes' : 'No'}</TableCell>
                <TableCell>{snippet.reason}</TableCell>
                {isAdmin && (
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Stack direction="row" spacing={1}>
                      <Tooltip title="Download snippet">
                        <Button
                          component="a"
                          href={buildSnippetDownloadUrl(snippet.filename)}
                          download
                          size="small"
                          startIcon={<DownloadIcon />}
                          onClick={() => markDownloaded(snippet.filename)}
                        >
                          Download
                        </Button>
                      </Tooltip>
                      <ApproveButton
                        snippet={snippet}
                        hasDownloaded={downloaded.has(snippet.filename)}
                      />
                    </Stack>
                  </TableCell>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </Box>
  );
}
