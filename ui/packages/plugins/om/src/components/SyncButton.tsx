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

import { useEffect, useRef, useState } from 'react';
import {
  Button,
  CircularProgress,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import SyncIcon from '@mui/icons-material/Sync';
import { enqueueSnackbar } from 'notistack';
import {
  useInvalidateOmTopologySnapshot,
  useOmTopologyRuns,
  useTriggerOmTopologyRun,
} from '../topologyHooks';
import { isRunActive, OmApiError } from '../api';

/**
 * Trigger a collection run and reflect its progress.
 *
 * Owns the whole interaction so both Overview and Inventory get identical
 * behaviour: disable while a run is in flight, poll (via `useOmTopologyRuns`) until it
 * reaches a terminal status, then invalidate the snapshot queries so the cluster
 * list reflects the new data without a manual refresh.
 *
 * The rebuild takes well under a second, too fast for the button's own state to show,
 * so the run it started is followed to its end and announced. Outlined, so it reads
 * lighter than Scan, which sends a job to every node.
 */
export const SyncButton = () => {
  const { data: runs } = useOmTopologyRuns();
  const trigger = useTriggerOmTopologyRun();
  const invalidateSnapshot = useInvalidateOmTopologySnapshot();
  const running = isRunActive(runs?.[0]?.status);
  const wasRunning = useRef(false);
  const [awaited, setAwaited] = useState<string | null>(null);
  const [runFailed, setRunFailed] = useState(false);

  useEffect(() => {
    if (wasRunning.current && !running) {
      invalidateSnapshot();
    }
    wasRunning.current = running;
  }, [running, invalidateSnapshot]);

  useEffect(() => {
    const run = runs?.find((candidate) => candidate.run_id === awaited);
    if (!run || isRunActive(run.status)) {
      return;
    }
    setAwaited(null);
    if (run.status === 'RUN_STATUS_FAILED') {
      setRunFailed(true);
    } else {
      enqueueSnackbar('Fleet updated', { variant: 'success' });
    }
  }, [runs, awaited]);

  // 409 means a run is already in flight — the expected answer to a double
  // click, not a failure. The transport carries the status onto the error so the
  // two can be told apart here.
  const conflict =
    trigger.error instanceof OmApiError && trigger.error.status === 409;
  const failure = trigger.error && !conflict ? (trigger.error as Error) : null;

  return (
    <Stack direction="row" alignItems="center" gap={1}>
      <Tooltip title="Re-read what PMM knows and rebuild this view. Does not scan your nodes.">
        <span>
          <Button
            variant="outlined"
            startIcon={running ? <CircularProgress size={16} /> : <SyncIcon />}
            disabled={running || trigger.isPending}
            onClick={() => {
              setRunFailed(false);
              trigger.mutate(undefined, {
                onSuccess: (accepted) => setAwaited(accepted.run_id),
              });
            }}
          >
            {running ? 'Refreshing…' : 'Refresh'}
          </Button>
        </span>
      </Tooltip>
      {conflict && (
        <Typography variant="body2" color="text.secondary">
          A refresh is already in flight.
        </Typography>
      )}
      {failure && (
        <Typography variant="body2" color="error">
          Could not refresh: {failure.message}
        </Typography>
      )}
      {runFailed && (
        <Typography variant="body2" color="error">
          The refresh did not finish, so this view still shows the data from
          before it.
        </Typography>
      )}
    </Stack>
  );
};
