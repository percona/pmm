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
import { describe, expect, it } from 'vitest';
import { StatusBadge } from '../StatusBadge';

describe('TaskLogViewer StatusBadge', () => {
  // The backend emits every finished status on the SSE `finish` event, so a
  // status missing from the map indexes it to `undefined` and throws while
  // reading `.color` -- the viewer goes blank rather than degrading.
  it.each([
    ['success', 'Done'],
    ['failed', 'Failed'],
    ['stopped', 'Stopped'],
    ['lost', 'Lost'],
    ['stale', 'Stale'],
    ['unlaunchable', 'Not in executor'],
    ['stream-error', 'Stream error'],
    ['executor-gone', 'Not in executor'],
  ] as const)('renders %s as %s', (status, label) => {
    render(<StatusBadge status={status} />);

    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it('renders a status it does not recognise instead of throwing', () => {
    // The union is a promise about the backend, not a check on it: the `finish`
    // event carries whatever the backend sends. Completing the map fixes the
    // statuses known today; this is what keeps the next one from blanking the
    // viewer before the union catches up.
    render(<StatusBadge status={'brand-new-status' as never} />);

    expect(screen.getByText('brand-new-status')).toBeInTheDocument();
  });
});
