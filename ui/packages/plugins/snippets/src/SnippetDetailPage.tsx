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

import { useNavigate, useParams } from 'react-router-dom';
import {
  Alert,
  Box,
  CircularProgress,
  IconButton,
  Link as MuiLink,
  Tooltip,
} from '@mui/material';
import DownloadIcon from '@mui/icons-material/Download';
import { SnippetExecutionAccordion } from '@sep/framework';
import { useSnippetDownload, useSnippetSchema } from './hooks';

/**
 * Snippet detail page rendered at `/snippets/:filename`.
 *
 * Delegates form rendering, execution, log streaming, and history display
 * to ``SnippetExecutionAccordion``. No ``executorHost`` prop is passed, so
 * the accordion renders the schema-driven host selector field.
 *
 * The legacy Jinja2 detail page remains mounted for capabilities not yet
 * ported (chaining, scheduling, alerting).
 */
export function SnippetDetailPage() {
  const { filename } = useParams<{ filename: string }>();
  const navigate = useNavigate();
  const schemaQuery = useSnippetSchema(filename, true);
  const downloadMutation = useSnippetDownload(filename);

  if (!filename) {
    return <Alert severity="error">Missing snippet filename in the URL.</Alert>;
  }

  const displayTitle = schemaQuery.data?.display_name ?? filename;
  const description = schemaQuery.data?.description;

  const downloadError = downloadMutation.isError
    ? (downloadMutation.error?.message ?? 'Download failed')
    : null;

  return (
    <Box>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          mb: 1,
        }}
      >
        <MuiLink
          component="button"
          type="button"
          onClick={() => navigate('..')}
          sx={{ display: 'inline-block' }}
        >
          ← Back to snippets
        </MuiLink>
        <Tooltip title={`Download ${filename}`}>
          <span>
            <IconButton
              aria-label={`Download ${filename}`}
              onClick={() => downloadMutation.mutate()}
              disabled={downloadMutation.isPending}
              size="small"
            >
              {downloadMutation.isPending ? (
                <CircularProgress size={20} />
              ) : (
                <DownloadIcon />
              )}
            </IconButton>
          </span>
        </Tooltip>
      </Box>

      {downloadError && (
        <Alert
          severity="error"
          sx={{ mb: 2 }}
          onClose={() => downloadMutation.reset()}
        >
          Failed to download snippet: {downloadError}
        </Alert>
      )}

      <SnippetExecutionAccordion
        snippetFilename={filename}
        title={displayTitle}
        description={description}
        defaultExpanded
        showHistory
      />
    </Box>
  );
}
