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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { LastSeenCell } from './LastSeenCell';

describe('LastSeenCell', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-11-23T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows the age alone while it is fresh', () => {
    render(<LastSeenCell lastSeenAt="2026-11-23T11:56:00Z" />);

    expect(screen.getByText('4m ago')).toBeInTheDocument();
    expect(screen.queryByTestId('om-last-seen-stale')).toBeNull();
  });

  it('adds a clock past five minutes, as the server calls a snapshot stale', () => {
    render(<LastSeenCell lastSeenAt="2026-11-23T10:00:00Z" />);

    expect(screen.getByText('2h ago')).toBeInTheDocument();
    expect(screen.getByTitle('Not observed for over 5m')).toBeInTheDocument();
  });

  it('says why there is no time when no member was observed', () => {
    render(<LastSeenCell lastSeenAt={null} />);

    expect(screen.queryByTestId('om-last-seen-stale')).toBeNull();
    expect(screen.queryByText(/ago/)).toBeNull();
  });
});
