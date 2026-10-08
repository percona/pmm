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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { browserTimezone } from '@pmm-extensions/framework';
import { SnapshotBar } from '../src/components/SnapshotBar';
import { topology } from './fixtures';

describe('SnapshotBar', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-12T09:03:00Z'));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows the snapshot age, and the local time and zone on hover', async () => {
    const envelope = topology([]).snapshot;
    render(<SnapshotBar envelope={envelope} />);

    await userEvent.hover(screen.getByText('Snapshot 3m ago'));

    expect(
      await screen.findByRole('tooltip', {
        name: `Snapshot generated ${new Date(envelope.generated_at!).toLocaleString()} (${browserTimezone()})`,
      })
    ).toBeInTheDocument();
  });
});
