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
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { EmptyState } from '../src/components/EmptyState';

const renderIt = (ui: React.ReactElement) =>
  render(<MemoryRouter>{ui}</MemoryRouter>);

describe('EmptyState', () => {
  it('says what the page shows and why it is empty', () => {
    renderIt(
      <EmptyState title="No scans yet">Nothing has run yet.</EmptyState>
    );

    expect(screen.getByText('No scans yet')).toBeInTheDocument();
    expect(screen.getByText('Nothing has run yet.')).toBeInTheDocument();
  });

  // The whole reason this component exists: an empty page should lead somewhere.
  it('offers the one action, when the page has one', () => {
    renderIt(
      <EmptyState
        title="No installs yet"
        action={{ label: 'Go to Nodes', to: '/om/nodes' }}
      >
        Start one from a node.
      </EmptyState>
    );

    expect(screen.getByRole('link', { name: 'Go to Nodes' })).toHaveAttribute(
      'href',
      '/om/nodes'
    );
  });

  // Not every page has an action -- a fleet with nothing in it needs services added
  // to PMM, which is not this app's to do. A button leading nowhere is worse.
  it('renders no action when the page has none', () => {
    renderIt(
      <EmptyState title="No nodes">PMM has none registered.</EmptyState>
    );

    expect(screen.queryByRole('link')).toBeNull();
  });
});
