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

import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { enqueueSnackbar, type VariantType } from 'notistack';
import { isRunActive, OmApiError } from './api';
import { pluralize } from './format';
import { useActiveInventoryRun, useOmInventoryRuns } from './inventoryHooks';
import { ScanTrackingContext } from './scanTracking';
import type { OmInventoryRun } from './types';

/** What a finished scan says about itself, and how loudly. */
export function scanLandedNotice(run: OmInventoryRun): {
  message: string;
  variant: VariantType;
} {
  if (run.status === 'RUN_STATUS_SKIPPED') {
    return {
      message: run.error ? `Scan skipped: ${run.error}` : 'Scan skipped',
      variant: 'info',
    };
  }
  if (run.status === 'RUN_STATUS_FAILED') {
    return {
      message: run.error ? `Scan failed: ${run.error}` : 'Scan failed',
      variant: 'error',
    };
  }
  const { answered_hosts: answered, probeable_hosts: probeable } = run.counts;
  if (answered >= probeable) {
    return {
      message: `Scan finished on ${answered} ${pluralize(answered, 'node')}`,
      variant: 'success',
    };
  }
  return {
    message: `Scan finished: ${answered} of ${probeable} nodes answered`,
    variant: 'warning',
  };
}

/**
 * Say when a scan started from these pages has finished, wherever the reader is by
 * then.
 *
 * Mounted once above every page, because a scan takes tens of seconds and the page
 * that started it may be gone. Only scans started here are announced: the schedule
 * starts one every ten minutes, and a notice for each would be noise.
 */
export const ScanFeedbackProvider = ({ children }: { children: ReactNode }) => {
  const [started, setStarted] = useState<string[]>([]);
  const track = useCallback(
    (runId: string) =>
      setStarted((ids) => (ids.includes(runId) ? ids : [...ids, runId])),
    []
  );
  const { data: runs } = useOmInventoryRuns({ enabled: started.length > 0 });

  useEffect(() => {
    const landed = (runs ?? []).filter(
      (run) => started.includes(run.run_id) && !isRunActive(run.status)
    );
    if (landed.length === 0) {
      return;
    }
    for (const run of landed) {
      const { message, variant } = scanLandedNotice(run);
      enqueueSnackbar(message, { variant });
    }
    setStarted((ids) =>
      ids.filter((id) => !landed.some((run) => run.run_id === id))
    );
  }, [runs, started]);

  return (
    <ScanTrackingContext.Provider value={track}>
      {children}
    </ScanTrackingContext.Provider>
  );
};

/**
 * A scan trigger's 409, kept only while the scan behind it is still running.
 *
 * Left in place, "a scan is already running" outlived that scan and read as a
 * standing fault. It is cleared once the history, read after the click, shows no scan
 * running. The running scan comes back too, so the notice can link to it.
 */
export function useScanConflict(trigger: {
  error: Error | null;
  submittedAt: number;
  reset: () => void;
}): { conflict: OmApiError | null; runningScan: OmInventoryRun | undefined } {
  const { run, updatedAt } = useActiveInventoryRun();
  const conflict =
    trigger.error instanceof OmApiError && trigger.error.status === 409
      ? trigger.error
      : null;
  const { reset, submittedAt } = trigger;

  useEffect(() => {
    if (conflict && !run && updatedAt > submittedAt) {
      reset();
    }
  }, [conflict, run, updatedAt, submittedAt, reset]);

  return { conflict, runningScan: conflict ? run : undefined };
}
