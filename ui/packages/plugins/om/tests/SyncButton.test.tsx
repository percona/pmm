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

import { act, fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SyncButton } from '../src/components/SyncButton';
import type { OmTopologyRun, OmTopologyRunStatus } from '../src/types';

const { enqueueSnackbar, useOmTopologyRuns, useTriggerOmTopologyRun, mutate } =
  vi.hoisted(() => ({
    enqueueSnackbar: vi.fn(),
    useOmTopologyRuns: vi.fn(),
    useTriggerOmTopologyRun: vi.fn(),
    mutate: vi.fn(),
  }));

vi.mock('notistack', () => ({ enqueueSnackbar }));
vi.mock('../src/topologyHooks', () => ({
  useOmTopologyRuns,
  useTriggerOmTopologyRun,
  useInvalidateOmTopologySnapshot: () => vi.fn(),
}));

const run = (run_id: string, status: OmTopologyRunStatus): OmTopologyRun => ({
  run_id,
  status,
  start_time: '2026-10-07T10:00:00Z',
  end_time: '2026-10-07T10:00:00.4Z',
  counts: {} as OmTopologyRun['counts'],
  errors: [],
});

/** Click Refresh and have the server accept it as `runId`. */
function clickRefresh(runId: string) {
  fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
  const [, options] = mutate.mock.calls[0];
  act(() => options.onSuccess({ run_id: runId }));
}

describe('SyncButton', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useOmTopologyRuns.mockReturnValue({ data: [] });
    useTriggerOmTopologyRun.mockReturnValue({
      mutate,
      isPending: false,
      error: null,
    });
  });

  it('reads lighter than Scan: outlined, not filled', () => {
    render(<SyncButton />);

    expect(screen.getByRole('button', { name: 'Refresh' })).toHaveClass(
      'MuiButton-outlined'
    );
  });

  it('says the fleet was updated once the run it started lands', () => {
    const { rerender } = render(<SyncButton />);
    clickRefresh('mine');
    expect(enqueueSnackbar).not.toHaveBeenCalled();

    useOmTopologyRuns.mockReturnValue({
      data: [run('mine', 'RUN_STATUS_PARTIAL')],
    });
    rerender(<SyncButton />);

    expect(enqueueSnackbar).toHaveBeenCalledWith('Fleet updated', {
      variant: 'success',
    });
  });

  it('keeps a failed run beside the button rather than in a notice', () => {
    const { rerender } = render(<SyncButton />);
    clickRefresh('mine');

    useOmTopologyRuns.mockReturnValue({
      data: [run('mine', 'RUN_STATUS_FAILED')],
    });
    rerender(<SyncButton />);

    expect(enqueueSnackbar).not.toHaveBeenCalled();
    expect(screen.getByText(/The refresh did not finish/)).toBeInTheDocument();
  });

  it('says nothing about a run it did not start', () => {
    useOmTopologyRuns.mockReturnValue({
      data: [run('timer', 'RUN_STATUS_SUCCESS')],
    });

    render(<SyncButton />);

    expect(enqueueSnackbar).not.toHaveBeenCalled();
  });
});
