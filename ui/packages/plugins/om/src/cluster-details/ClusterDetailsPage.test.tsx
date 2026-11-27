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

import { fireEvent, render, screen, within } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { OmTopologyResponse } from '../types';
import {
  cluster,
  replicaSetCluster,
  service,
  shardedCluster,
  topology,
} from '../../tests/fixtures';
import { LocationProbe, PmmTheme } from '../../tests/testUtils';
import { ClusterDetailsPage } from './ClusterDetailsPage';

const { useOmTopology } = vi.hoisted(() => ({ useOmTopology: vi.fn() }));

vi.mock('../topologyHooks', () => ({ useOmTopology }));
vi.mock('../inventoryHooks', () => ({
  useLastScanFinishedAt: () => ({ status: 'ready', finishedAt: null }),
}));

const loaded = (data: OmTopologyResponse) =>
  useOmTopology.mockReturnValue({
    data,
    isPending: false,
    isError: false,
    error: null,
    dataUpdatedAt: 0,
  });

const estate = () =>
  topology([
    {
      env_name: 'production',
      clusters: [
        replicaSetCluster({ id: 'rs-id' }),
        shardedCluster({ id: 'sharded-id' }),
        cluster({
          id: 'solo-id',
          name: 'solo',
          type: 'CLUSTER_TYPE_STANDALONE',
          services: [service({ service_name: 'solo-1', endpoint: 'solo:1' })],
        }),
      ],
    },
  ]);

const renderPage = (clusterId: string) =>
  render(
    <MemoryRouter initialEntries={[`/operations/clusters/${clusterId}`]}>
      <Routes>
        <Route
          path="/operations/clusters/:clusterId"
          element={
            <>
              <ClusterDetailsPage />
              <LocationProbe />
            </>
          }
        />
        <Route path="*" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>,
    { wrapper: PmmTheme }
  );

const memberRows = () =>
  within(screen.getByTestId('om-members-section'))
    .getAllByRole('row')
    .filter((row) => within(row).queryByTestId('om-service-status'));

describe('ClusterDetailsPage', () => {
  beforeEach(() => {
    loaded(estate());
  });

  it('shows progress while the snapshot loads', () => {
    useOmTopology.mockReturnValue({ isPending: true });
    renderPage('rs-id');

    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('says why when the snapshot cannot be read', () => {
    useOmTopology.mockReturnValue({
      isPending: false,
      isError: true,
      error: new Error('boom'),
    });
    renderPage('rs-id');

    expect(screen.getByText('Could not load the cluster')).toBeInTheDocument();
    expect(screen.getByText('boom')).toBeInTheDocument();
  });

  it('leads back to Fleet for an id no cluster has', () => {
    renderPage('gone');

    expect(screen.getByText('Cluster not found')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to Fleet' })).toHaveAttribute(
      'href',
      '/operations'
    );
  });

  it('heads a replica set with its name, health and facts', () => {
    renderPage('rs-id');

    expect(
      screen.getByRole('heading', { level: 1, name: 'reporting' })
    ).toBeInTheDocument();
    expect(screen.getByTestId('om-cluster-health')).toHaveTextContent(
      'Healthy'
    );
    const chips = screen.getByTestId('om-cluster-chips');
    expect(chips).toHaveTextContent('Replica set');
    expect(chips).toHaveTextContent('rs-reporting');
    expect(chips).toHaveTextContent('3 members');
    expect(chips).toHaveTextContent('7.0.39-21');
    expect(chips).toHaveTextContent('production');
  });

  it('counts a sharded cluster’s shards beside its members', () => {
    renderPage('sharded-id');

    expect(screen.getByTestId('om-cluster-chips')).toHaveTextContent(
      '2 shards · 5 members'
    );
  });

  it('links to the cluster’s monitoring dashboard', () => {
    renderPage('rs-id');

    expect(
      screen.getByRole('link', { name: 'Monitoring dashboard' })
    ).toHaveAttribute(
      'href',
      '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=production&var-cluster=reporting&var-rs_nm=rs-reporting'
    );
  });

  it('offers the connection string to copy', () => {
    renderPage('rs-id');

    expect(screen.getByTestId('om-connection-string-value')).toHaveTextContent(
      'mongodb://rpt-1.eu:27017,rpt-3.eu:27017,rpt-2.eu:27017/?replicaSet=rs-reporting'
    );
    expect(
      within(screen.getByTestId('om-connection-string')).getByRole('button', {
        name: /copy/i,
      })
    ).toBeInTheDocument();
  });

  it('shows no cluster actions while none can run', () => {
    renderPage('rs-id');

    expect(
      screen.queryByRole('button', { name: 'Cluster actions' })
    ).toBeNull();
  });

  describe('switcher', () => {
    const open = () =>
      fireEvent.click(screen.getByRole('button', { name: 'Switch cluster' }));

    it('lists every cluster by name and marks the current one', () => {
      renderPage('rs-id');
      open();

      const items = screen.getAllByRole('menuitem');
      expect(items.map((item) => item.textContent)).toEqual([
        'HealthyordersSharded · production',
        'HealthyreportingReplica set · production',
        'HealthysoloStandalone · production',
      ]);
      expect(items[1]).toHaveClass('Mui-selected');
    });

    it('narrows the list as the reader types', () => {
      renderPage('rs-id');
      open();
      fireEvent.change(
        screen.getByRole('textbox', { name: 'Find a cluster' }),
        {
          target: { value: 'ORD' },
        }
      );

      expect(screen.getAllByRole('menuitem')).toHaveLength(1);
      fireEvent.change(
        screen.getByRole('textbox', { name: 'Find a cluster' }),
        {
          target: { value: 'nothing' },
        }
      );
      expect(screen.getByText('No clusters match')).toBeInTheDocument();
    });

    it('opens the picked cluster', () => {
      renderPage('rs-id');
      open();
      fireEvent.click(screen.getByRole('menuitem', { name: /orders/ }));

      expect(screen.getByTestId('location')).toHaveTextContent(
        '/operations/clusters/sharded-id'
      );
    });
  });

  describe('members', () => {
    it('sums up how many are reachable', () => {
      renderPage('rs-id');

      expect(screen.getByTestId('om-members-reachable')).toHaveTextContent(
        '3 / 3 reachable'
      );
      expect(screen.getByText('all healthy')).toBeInTheDocument();
    });

    it('lists the primary first, with its role and no lag of its own', () => {
      renderPage('rs-id');

      const rows = memberRows();
      expect(rows.map((row) => row.textContent)).toEqual([
        expect.stringContaining('rpt-1.eu'),
        expect.stringContaining('rpt-2.eu'),
        expect.stringContaining('rpt-3.eu'),
      ]);
      expect(rows[0]).toHaveTextContent('Primary');
      expect(rows[1]).toHaveTextContent('Secondary');
      expect(within(rows[2]).getByTestId('om-threshold-error')).toBeVisible();
    });

    it('flags an oplog window below an hour', () => {
      renderPage('rs-id');

      expect(
        within(memberRows()[0]).getByTestId('om-threshold-warning')
      ).toBeVisible();
    });

    it('opens a member’s dashboards from its menu, and offers nothing else yet', () => {
      renderPage('rs-id');
      fireEvent.click(
        screen.getByRole('button', { name: 'Actions for rpt-1.eu' })
      );

      const items = screen.getAllByRole('menuitem');
      expect(items).toHaveLength(1);
      expect(items[0]).toHaveTextContent('Open in PMM dashboards');
      expect(items[0]).toHaveAttribute(
        'href',
        '/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-service_name=rpt-1'
      );
    });

    it('groups a sharded cluster into routers, config servers and shards', () => {
      renderPage('sharded-id');

      expect(
        screen
          .getAllByTestId('om-member-group')
          .map((group) => group.textContent)
      ).toEqual([
        'Routers2 members',
        'Config servers1 member',
        'rs-a1 member',
        'rs-b1 member',
      ]);
    });
  });
});
