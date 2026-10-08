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

import { act, render, renderHook, screen } from '@testing-library/react';
import { useContext } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OmApiError } from '../src/api';
import {
  ScanFeedbackProvider,
  scanLandedNotice,
  useScanConflict,
} from '../src/ScanFeedback';
import { ScanTrackingContext } from '../src/scanTracking';
import type { OmInventoryRun, OmTopologyRunStatus } from '../src/types';

const { enqueueSnackbar, useOmInventoryRuns, useActiveInventoryRun } =
  vi.hoisted(() => ({
    enqueueSnackbar: vi.fn(),
    useOmInventoryRuns: vi.fn(),
    useActiveInventoryRun: vi.fn(),
  }));

vi.mock('notistack', () => ({ enqueueSnackbar }));
vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryRuns,
  useActiveInventoryRun,
}));

const run = (
  run_id: string,
  status: OmTopologyRunStatus,
  answered = 3,
  probeable = 3
): OmInventoryRun => ({
  run_id,
  status,
  start_time: '2026-10-07T10:00:00Z',
  end_time: status === 'RUN_STATUS_RUNNING' ? null : '2026-10-07T10:00:40Z',
  scope: [],
  counts: {
    total_services: 0,
    resolved_services: 0,
    orphaned_services: 0,
    answered_services: 0,
    total_hosts: probeable,
    probeable_hosts: probeable,
    answered_hosts: answered,
  },
});

describe('scanLandedNotice', () => {
  it('counts the nodes a clean scan finished on', () => {
    expect(scanLandedNotice(run('r', 'RUN_STATUS_SUCCESS', 3, 3))).toEqual({
      message: 'Scan finished on 3 nodes',
      variant: 'success',
    });
    expect(scanLandedNotice(run('r', 'RUN_STATUS_SUCCESS', 1, 1)).message).toBe(
      'Scan finished on 1 node'
    );
  });

  it('says how many answered when some did not', () => {
    expect(scanLandedNotice(run('r', 'RUN_STATUS_PARTIAL', 7, 12))).toEqual({
      message: 'Scan finished: 7 of 12 nodes answered',
      variant: 'warning',
    });
  });

  it('says a failed run failed, with its reason', () => {
    expect(
      scanLandedNotice({ ...run('r', 'RUN_STATUS_FAILED'), error: 'boom' })
    ).toEqual({ message: 'Scan failed: boom', variant: 'error' });
  });

  it('says why a run was skipped, as information', () => {
    expect(
      scanLandedNotice({
        ...run('r', 'RUN_STATUS_SKIPPED'),
        error: 'Scanning is switched off',
      })
    ).toEqual({
      message: 'Scan skipped: Scanning is switched off',
      variant: 'info',
    });
  });
});

describe('ScanFeedbackProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  /** Report one scan as started from a page, the way useRefreshInventory does. */
  const Starter = ({ runId }: { runId: string }) => {
    const track = useContext(ScanTrackingContext);
    return (
      <button type="button" onClick={() => track?.(runId)}>
        start
      </button>
    );
  };

  it('announces a scan started here once it lands, and only once', () => {
    useOmInventoryRuns.mockReturnValue({
      data: [run('mine', 'RUN_STATUS_RUNNING')],
    });
    const { rerender } = render(
      <ScanFeedbackProvider>
        <Starter runId="mine" />
      </ScanFeedbackProvider>
    );
    act(() => screen.getByRole('button', { name: 'start' }).click());
    expect(enqueueSnackbar).not.toHaveBeenCalled();

    useOmInventoryRuns.mockReturnValue({
      data: [run('mine', 'RUN_STATUS_SUCCESS', 2, 2)],
    });
    rerender(
      <ScanFeedbackProvider>
        <Starter runId="mine" />
      </ScanFeedbackProvider>
    );
    rerender(
      <ScanFeedbackProvider>
        <Starter runId="mine" />
      </ScanFeedbackProvider>
    );

    expect(enqueueSnackbar).toHaveBeenCalledTimes(1);
    expect(enqueueSnackbar).toHaveBeenCalledWith('Scan finished on 2 nodes', {
      variant: 'success',
    });
  });

  it('stays quiet about scans the schedule started', () => {
    useOmInventoryRuns.mockReturnValue({
      data: [run('scheduled', 'RUN_STATUS_SUCCESS')],
    });

    render(
      <ScanFeedbackProvider>
        <Starter runId="mine" />
      </ScanFeedbackProvider>
    );

    expect(enqueueSnackbar).not.toHaveBeenCalled();
    // Nothing started here, so the history is not even read for this.
    expect(useOmInventoryRuns).toHaveBeenLastCalledWith({ enabled: false });
  });
});

describe('useScanConflict', () => {
  const conflict = new OmApiError(409, 'A scan is already running on node00');

  it('keeps the conflict, and the scan to link to, while that scan runs', () => {
    const running = run('other', 'RUN_STATUS_RUNNING');
    useActiveInventoryRun.mockReturnValue({ run: running, updatedAt: 20 });
    const reset = vi.fn();

    const { result } = renderHook(() =>
      useScanConflict({ error: conflict, submittedAt: 10, reset })
    );

    expect(result.current.conflict).toBe(conflict);
    expect(result.current.runningScan).toBe(running);
    expect(reset).not.toHaveBeenCalled();
  });

  it('clears it once the history, read after the click, shows no scan', () => {
    useActiveInventoryRun.mockReturnValue({ run: undefined, updatedAt: 20 });
    const reset = vi.fn();

    renderHook(() =>
      useScanConflict({ error: conflict, submittedAt: 10, reset })
    );

    expect(reset).toHaveBeenCalledTimes(1);
  });

  it('waits for a read newer than the click before clearing', () => {
    useActiveInventoryRun.mockReturnValue({ run: undefined, updatedAt: 5 });
    const reset = vi.fn();

    renderHook(() =>
      useScanConflict({ error: conflict, submittedAt: 10, reset })
    );

    expect(reset).not.toHaveBeenCalled();
  });

  it('leaves any other error alone', () => {
    useActiveInventoryRun.mockReturnValue({ run: undefined, updatedAt: 20 });
    const reset = vi.fn();

    const { result } = renderHook(() =>
      useScanConflict({
        error: new OmApiError(500, 'boom'),
        submittedAt: 10,
        reset,
      })
    );

    expect(result.current.conflict).toBeNull();
    expect(reset).not.toHaveBeenCalled();
  });
});
