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

import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { ServiceDetailDrawer } from '../src/components/ServiceDetailDrawer';
import type { OmInventoryService, OmServiceInventoryRow } from '../src/types';
import { service } from './fixtures';

/** A scan row for the service, with the freshness `ProbeValue` reads. */
const scanned = (
  overrides: Partial<OmInventoryService> = {}
): OmInventoryService => ({
  service_id: '30',
  node_id: 'node-uuid-1',
  observed: {},
  freshness: {
    last_success_at: '2026-10-06T09:00:00Z',
    consecutive_failures: 0,
  },
  ...overrides,
});

const row = (
  overrides: Partial<OmServiceInventoryRow> = {}
): OmServiceInventoryRow => ({
  ...service({}),
  env_name: 'production',
  cluster_name: 'orders',
  inventory: null,
  ...overrides,
});

const renderDrawer = (
  props: Partial<React.ComponentProps<typeof ServiceDetailDrawer>> = {}
) =>
  render(
    <MemoryRouter>
      <ServiceDetailDrawer
        row={row()}
        estate="ready"
        onClose={vi.fn()}
        {...props}
      />
    </MemoryRouter>
  );

describe('ServiceDetailDrawer', () => {
  it('renders nothing until a row is selected', () => {
    renderDrawer({ row: null });

    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('is a dialog named by the service it describes', () => {
    renderDrawer();

    expect(screen.getByRole('dialog')).toHaveAccessibleName('svc');
  });

  // The whole reason it exists: these are the columns the short table drops, and
  // the column chooser can only bring them back for every row at once.
  it('shows the fields the table does not', () => {
    renderDrawer({
      row: row({
        endpoint: 'node00:27017',
        edition: 'Community',
        service_id: '30',
        inventory: scanned({
          config_path: '/etc/mongod.conf',
          argv: '/usr/bin/mongod --config /etc/mongod.conf --replSet rs0',
        }),
      }),
    });

    expect(screen.getByText('node00:27017')).toBeInTheDocument();
    expect(screen.getByText('Community')).toBeInTheDocument();
    expect(screen.getByText('30')).toBeInTheDocument();
    expect(screen.getByText('/etc/mongod.conf')).toBeInTheDocument();
    expect(
      screen.getByText(
        '/usr/bin/mongod --config /etc/mongod.conf --replSet rs0'
      )
    ).toBeInTheDocument();
  });

  // P17's last bullet: a detail view links out to where the rest of PMM
  // describes the same service.
  it('links to the service elsewhere in PMM', () => {
    renderDrawer();

    expect(screen.getByRole('link')).toBeInTheDocument();
  });

  it('closes on the close button', () => {
    const onClose = vi.fn();
    renderDrawer({ onClose });

    fireEvent.click(screen.getByLabelText('Close service details'));

    expect(onClose).toHaveBeenCalledOnce();
  });
});
