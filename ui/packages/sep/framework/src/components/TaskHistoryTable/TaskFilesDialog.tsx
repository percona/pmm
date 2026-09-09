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

import { useEffect } from 'react';
import DownloadIcon from '@mui/icons-material/Download';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useTaskHistoryFiles } from '../../hooks/useTaskHistoryFiles';
import { useTaskFileDownload } from '../../hooks/useTaskFileDownload';

function formatSize(size: number): string {
  if (size === 0) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.min(
    Math.floor(Math.log(size) / Math.log(1024)),
    units.length - 1
  );
  return `${(size / 1024 ** i).toFixed(1)} ${units[i]}`;
}

function basename(path: string): string {
  return path.split('/').filter(Boolean).pop() ?? path;
}

export interface TaskFilesDialogProps {
  open: boolean;
  taskHistoryId: number | null | undefined;
  onClose: () => void;
}

export function TaskFilesDialog({
  open,
  taskHistoryId,
  onClose,
}: TaskFilesDialogProps) {
  const { data, isLoading, isError } = useTaskHistoryFiles(
    open ? taskHistoryId : null
  );
  const downloadMutation = useTaskFileDownload();
  const { reset: resetMutation } = downloadMutation;

  useEffect(() => {
    if (!open) {
      resetMutation();
    }
  }, [open, resetMutation]);

  const entries = data ? Object.entries(data) : [];

  function handleDownload(path: string, isDir: boolean) {
    if (taskHistoryId === null || taskHistoryId === undefined) {
      return;
    }
    downloadMutation.mutate({ taskHistoryId, path, isDir });
  }

  const anyPending = downloadMutation.isPending;

  return (
    <Dialog
      open={open}
      onClose={onClose}
      fullWidth
      maxWidth="sm"
      aria-labelledby="task-files-dialog-title"
    >
      <DialogTitle id="task-files-dialog-title">Download files</DialogTitle>
      <DialogContent>
        {isLoading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
            <CircularProgress size={32} />
          </Box>
        )}
        {isError && <Alert severity="error">Failed to load file list.</Alert>}
        {downloadMutation.error && (
          <Alert
            severity="error"
            onClose={() => resetMutation()}
            sx={{ mb: 1 }}
          >
            {downloadMutation.error.message || 'Download failed'}
          </Alert>
        )}
        {!isLoading && !isError && entries.length === 0 && (
          <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>
            No files available for this task run.
          </Typography>
        )}
        {!isLoading && !isError && entries.length > 0 && (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Size</TableCell>
                <TableCell />
              </TableRow>
            </TableHead>
            <TableBody>
              {entries.map(([path, meta]) => {
                const name = basename(path);
                const isThisRowDownloading =
                  anyPending && downloadMutation.variables?.path === path;
                return (
                  <TableRow key={path}>
                    <TableCell sx={{ wordBreak: 'break-all' }} title={path}>
                      {name}
                    </TableCell>
                    <TableCell>
                      {meta.is_dir ? 'Folder' : formatSize(meta.size)}
                    </TableCell>
                    <TableCell align="right">
                      <Tooltip
                        title={
                          meta.is_dir
                            ? `Download as ${name}.tar.gz`
                            : 'Download'
                        }
                      >
                        <span>
                          <IconButton
                            size="small"
                            aria-label={`Download ${path}`}
                            disabled={anyPending}
                            aria-busy={isThisRowDownloading}
                            onClick={() => handleDownload(path, meta.is_dir)}
                          >
                            <DownloadIcon fontSize="small" />
                          </IconButton>
                        </span>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </DialogContent>
    </Dialog>
  );
}
