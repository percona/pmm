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

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { browserTimezone } from '@pmm-extensions/framework';
import { SnapshotBar } from '../src/components/SnapshotBar';

const { useLastScanFinishedAt } = vi.hoisted(() => ({
  useLastScanFinishedAt: vi.fn(),
}));

vi.mock('../src/inventoryHooks', () => ({ useLastScanFinishedAt }));

const minutesAgo = (minutes: number) =>
  new Date(Date.now() - minutes * 60_000).toISOString();

const envelope = {
  generated_at: minutesAgo(0),
  observed_at: minutesAgo(0),
  stale: false,
  schema_version: 1,
  run_id: 'run',
};

describe('SnapshotBar', () => {
  beforeEach(() => {
    useLastScanFinishedAt.mockReturnValue({
      status: 'ready',
      finishedAt: minutesAgo(9),
    });
  });

  it('says how old the data is and when the nodes were last scanned', () => {
    const { container } = render(<SnapshotBar envelope={envelope} />);

    expect(screen.getByText(/^Data from \d+s ago$/)).toBeInTheDocument();
    expect(screen.getByText('Nodes last scanned 9m ago')).toBeInTheDocument();
    // Pedro's wording, 2026-10-06: none of the internal words.
    expect(container).not.toHaveTextContent(/snapshot|observation|probe/i);
  });

  it('says the nodes have not been scanned rather than drawing a dash', () => {
    useLastScanFinishedAt.mockReturnValue({
      status: 'ready',
      finishedAt: null,
    });

    render(<SnapshotBar envelope={envelope} />);

    expect(screen.getByText('Nodes not scanned yet')).toBeInTheDocument();
  });

  it('says the scan history is unavailable rather than that nothing was scanned', () => {
    useLastScanFinishedAt.mockReturnValue({
      status: 'unavailable',
      finishedAt: null,
    });

    render(<SnapshotBar envelope={envelope} />);

    expect(screen.getByText('Scan history unavailable')).toBeInTheDocument();
    expect(screen.queryByText('Nodes not scanned yet')).not.toBeInTheDocument();
  });

  it('says the scan history is being read before it has answered', () => {
    useLastScanFinishedAt.mockReturnValue({
      status: 'pending',
      finishedAt: null,
    });

    render(<SnapshotBar envelope={envelope} />);

    expect(screen.getByText('Reading scan history…')).toBeInTheDocument();
    expect(screen.queryByText('Nodes not scanned yet')).not.toBeInTheDocument();
  });

  it('puts the local time and zone of the data on hover', async () => {
    render(<SnapshotBar envelope={envelope} />);

    await userEvent.hover(screen.getByText(/^Data from/));

    const local = (iso: string) =>
      `${new Date(iso).toLocaleString()} (${browserTimezone()})`;
    expect(
      await screen.findByRole('tooltip', {
        name: `Assembled ${local(envelope.generated_at)}. Newest metric or scan reading in it: ${local(envelope.observed_at)}.`,
      })
    ).toBeInTheDocument();
  });
});
