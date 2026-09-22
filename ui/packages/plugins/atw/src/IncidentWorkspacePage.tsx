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

import { useCallback, useRef, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Link as MuiLink,
  Paper,
  Stack,
  Typography,
} from '@mui/material';
import LockOpenOutlinedIcon from '@mui/icons-material/LockOpenOutlined';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { useNavigate, useParams } from 'react-router-dom';
import { useAuth } from '@sep/api';
import { ReadOnlyNotice } from '@sep/framework';
import { CollectPane } from './CollectPane';
import { ResultsPane } from './ResultsPane';
import { useAtwIncident, useAtwIncidentLifecycle } from './hooks';
import type {
  AtwBatchExecuteResponse,
  AtwIncidentExecution,
  AtwRememberedDispatch,
  AtwRerunRequest,
  AtwSnippetSummary,
} from './types';

/**
 * Incident workspace rendered at ``/atw/:incidentId``. Two side-by-side panes —
 * Collect (browse, select, and batch-execute snippets) and Results (each
 * execution's status, logs, and file listing) — stacked on narrow screens.
 *
 * A read-only session gets Results alone, full width. Collect exists only to
 * start an execution, and every unsafe ATW route requires an administrator, so
 * leaving the pane mounted with its execute form withheld offered a snippet
 * picker that could never run anything. Withheld rather than disabled because a
 * role, unlike a closed incident, is not something the viewer can undo.
 */
export function IncidentWorkspacePage() {
  const { incidentId } = useParams<{ incidentId: string }>();
  const navigate = useNavigate();
  const { canMutate } = useAuth();
  const { data: incident, isLoading, error } = useAtwIncident(incidentId);
  const lifecycle = useAtwIncidentLifecycle();
  const isClosed = Boolean(incident?.closed_at);

  // What this browser tab has dispatched, so a past execution's "Run again"
  // and "Edit parameters and run again" have something to act on — nothing
  // re-runnable rides on the wire, so a reload or another tab's execution has
  // no entry here and falls back to a snippet-only prefill (see CollectPane).
  const [remembered, setRemembered] = useState<
    Map<number, AtwRememberedDispatch>
  >(new Map());
  const [rerunRequest, setRerunRequest] = useState<AtwRerunRequest | null>(
    null
  );
  const rerunNonceRef = useRef(0);
  const collectSectionRef = useRef<HTMLDivElement>(null);

  const handleDispatched = useCallback(
    (
      snippets: AtwSnippetSummary[],
      values: Record<string, unknown>,
      response: AtwBatchExecuteResponse
    ) => {
      setRemembered((previous) => {
        const next = new Map(previous);
        for (const item of response.items) {
          if (
            item.task_history_id !== null &&
            item.task_history_id !== undefined
          ) {
            next.set(item.task_history_id, { snippets, values });
          }
        }
        return next;
      });
    },
    []
  );

  const handleEditParameters = useCallback(
    (execution: AtwIncidentExecution) => {
      rerunNonceRef.current += 1;
      setRerunRequest({
        nonce: rerunNonceRef.current,
        snippetFilename: execution.snippet_filename,
        remembered: remembered.get(execution.task_history_id),
      });
      const target = collectSectionRef.current;
      // jsdom (and some embeds) do not implement this API at all — matches
      // the same guard `SchemaFormRenderer` uses for its own scroll-into-view.
      if (target && typeof target.scrollIntoView === 'function') {
        target.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    },
    [remembered]
  );

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

      {lifecycle.error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={lifecycle.reset}>
          {lifecycle.error}
        </Alert>
      )}

      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        alignItems={{ xs: 'flex-start', sm: 'center' }}
        justifyContent="space-between"
        spacing={1}
        sx={{ mb: 0.5 }}
      >
        <Stack
          direction="row"
          alignItems="center"
          spacing={1}
          flexWrap="wrap"
          useFlexGap
        >
          <Typography variant="h4">{incident?.name ?? 'Incident'}</Typography>
          {isClosed && (
            <Chip
              label="Closed"
              size="small"
              color="default"
              variant="outlined"
            />
          )}
        </Stack>
        {incident && canMutate && (
          <Stack direction="row" spacing={1}>
            {isClosed ? (
              <Button
                variant="outlined"
                size="small"
                startIcon={<LockOpenOutlinedIcon />}
                disabled={lifecycle.isPending(incident.id)}
                onClick={() => lifecycle.reopen(incident.id)}
              >
                Reopen incident
              </Button>
            ) : (
              <Button
                variant="outlined"
                startIcon={<LockOutlinedIcon />}
                disabled={lifecycle.isPending(incident.id)}
                onClick={() => lifecycle.close(incident.id)}
              >
                Close incident
              </Button>
            )}
          </Stack>
        )}
      </Stack>
      {incident?.case_ref && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          Case reference: {incident.case_ref}
        </Typography>
      )}

      {!canMutate && (
        <Box sx={{ mt: 1, mb: 2 }}>
          <ReadOnlyNotice
            variant="inline"
            action="collect diagnostics for this incident"
            testId="atw-collect-read-only"
          />
        </Box>
      )}

      <Box
        sx={{
          mt: 1,
          display: 'grid',
          gap: 2,
          gridTemplateColumns: {
            xs: 'minmax(0, 1fr)',
            md: canMutate ? 'repeat(2, minmax(0, 1fr))' : 'minmax(0, 1fr)',
          },
          alignItems: 'start',
        }}
      >
        {/* PMM divergence from upstream SEP — keep on the next sync. */}
        {canMutate && (
          <Paper variant="outlined" sx={{ p: 2 }} ref={collectSectionRef}>
            <CollectPane
              incidentId={incidentId}
              isClosed={isClosed}
              rerunRequest={rerunRequest}
              onDispatched={handleDispatched}
            />
          </Paper>
        )}
        <Paper variant="outlined" sx={{ p: 2 }}>
          <ResultsPane
            incidentId={incidentId}
            remembered={remembered}
            onDispatched={handleDispatched}
            onEditParameters={handleEditParameters}
          />
        </Paper>
      </Box>
    </Box>
  );
}
