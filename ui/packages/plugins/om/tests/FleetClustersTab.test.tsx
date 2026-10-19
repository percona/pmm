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
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { FLEET_NOT_COLLECTED } from '../src/constants';
import { FleetClustersTab } from '../src/FleetClustersTab';
import { cluster, mixedEstate, service, topology } from './fixtures';

const { useOmTopology } = vi.hoisted(() => ({ useOmTopology: vi.fn() }));

vi.mock('../src/topologyHooks', () => ({ useOmTopology }));
vi.mock('../src/inventoryHooks', () => ({
  useLastScanFinishedAt: () => ({ status: 'ready', finishedAt: null }),
}));

const renderPage = (url = '/operations') =>
  render(
    <MemoryRouter initialEntries={[url]}>
      <FleetClustersTab />
    </MemoryRouter>
  );

const clusterRows = () =>
  screen
    .getAllByRole('row')
    .filter((row) => within(row).queryByTestId('om-cluster-health'));

describe('FleetClustersTab', () => {
  beforeEach(() => {
    useOmTopology.mockReturnValue({
      data: mixedEstate(),
      isPending: false,
      isError: false,
    });
  });

  it('names each cluster state and leads with the degraded one', () => {
    renderPage();

    const [first, second] = clusterRows();
    expect(within(first).getByTestId('om-cluster-health')).toHaveTextContent(
      'Degraded'
    );
    expect(first).toHaveTextContent('orders');
    expect(within(second).getByTestId('om-cluster-health')).toHaveTextContent(
      'Healthy'
    );
  });

  it('labels the process type "Process", never "Role"', () => {
    renderPage();

    expect(
      screen.getByRole('columnheader', { name: /Process/ })
    ).toBeInTheDocument();
    expect(screen.queryByRole('columnheader', { name: /Role/ })).toBeNull();
  });

  it('lists an unfolded cluster down member first, with member state as text', () => {
    renderPage();

    fireEvent.click(
      within(clusterRows()[0]).getByRole('button', { name: /expand/i })
    );

    const members = screen
      .getAllByRole('row')
      // The detail panel's own row wraps the whole member table; skip it.
      .filter(
        (row) => within(row).queryAllByTestId('om-service-status').length === 1
      );
    expect(
      members.map((row) => within(row).getByRole('link').textContent)
    ).toEqual(['orders-2', 'orders-1', 'orders-3', 'orders-router']);
    expect(members[0]).toHaveTextContent('SECONDARY');
    expect(members[1]).toHaveTextContent('PRIMARY');
    // A router is no replica-set member: its process says so, not a state.
    expect(members[3]).toHaveTextContent('Router');
    expect(members[3]).not.toHaveTextContent(/PRIMARY|SECONDARY/);
    expect(
      screen.getByRole('columnheader', { name: 'Member state' })
    ).toBeInTheDocument();
  });

  it('says beside a down member how long it has been down', () => {
    useOmTopology.mockReturnValue({
      data: topology([
        {
          env_name: 'production',
          clusters: [
            cluster({
              name: 'orders',
              services: [
                service({ service_name: 'orders-1', state: 'PRIMARY' }),
                service({
                  service_name: 'orders-2',
                  status: 'SERVICE_STATUS_DOWN',
                  last_up_at: new Date(
                    Date.now() - 2 * 3600 * 1000
                  ).toISOString(),
                }),
              ],
            }),
          ],
        },
      ]),
      isPending: false,
      isError: false,
    });
    renderPage();

    fireEvent.click(
      within(clusterRows()[0]).getByRole('button', { name: /expand/i })
    );

    const down = screen
      .getAllByRole('row')
      .find((row) => within(row).queryByText('orders-2')) as HTMLElement;
    expect(within(down).getByTestId('om-down-for')).toHaveTextContent(
      /^for 2h( \d+s)?$/
    );
  });

  it('links each member to its PMM dashboard, a down one included', () => {
    renderPage();

    fireEvent.click(
      within(clusterRows()[0]).getByRole('button', { name: /expand/i })
    );

    expect(screen.getByRole('link', { name: 'orders-2' })).toHaveAttribute(
      'href',
      expect.stringContaining('var-service_name=orders-2')
    );
  });

  it('blames PMM for an empty fleet only once the snapshot has been built', () => {
    useOmTopology.mockReturnValue({
      data: topology([]),
      isPending: false,
      isError: false,
    });
    renderPage();

    expect(screen.getByTestId('om-empty-state')).toHaveTextContent(
      'PMM has no MongoDB services registered yet'
    );
  });

  it('says the fleet has not been read yet before the first snapshot', () => {
    const cold = topology([]);
    cold.snapshot.generated_at = undefined;
    useOmTopology.mockReturnValue({
      data: cold,
      isPending: false,
      isError: false,
    });
    renderPage();

    const empty = screen.getByTestId('om-empty-state');
    expect(empty).toHaveTextContent(FLEET_NOT_COLLECTED);
    expect(empty).not.toHaveTextContent('PMM has no MongoDB services');
  });

  describe('counts carry no status colour (Pedro, 2026-10-06 and 2026-10-07)', () => {
    // MUI's default palette, which these tests render under.
    const RED = 'rgb(211, 47, 47)';
    const GREEN = 'rgb(46, 125, 50)';
    const colourOf = (element: HTMLElement) => getComputedStyle(element).color;
    // The number is in a <strong> of its own, so match the line it sits in.
    const line = (text: string) =>
      screen.getAllByText((_, element) => element?.textContent === text)[0];

    it('keeps "up" neutral on the summary lines, and "down" red only above zero', () => {
      renderPage();

      for (const up of screen.getAllByText(/\bup$/)) {
        expect([RED, GREEN]).not.toContain(colourOf(up));
      }
      expect(colourOf(line('1 down'))).toBe(RED);
      expect(colourOf(line('0 down'))).not.toBe(RED);
    });

    it("states a cluster's members up as plain text, in every state", () => {
      const down = { status: 'SERVICE_STATUS_DOWN' } as const;
      useOmTopology.mockReturnValue({
        data: topology([
          {
            env_name: 'production',
            clusters: [
              cluster({
                name: 'mixed',
                services: [
                  service({ service_name: 'mixed-1' }),
                  service({ service_name: 'mixed-2', ...down }),
                ],
              }),
              cluster({ name: 'empty', services: [] }),
              cluster({
                name: 'dark',
                services: [service({ service_name: 'dark-1', ...down })],
              }),
            ],
          },
        ]),
        isPending: false,
        isError: false,
      });
      renderPage();

      // Hidden by default; shown the way a reader would.
      fireEvent.click(
        screen.getByRole('button', { name: /show\/hide columns/i })
      );
      const upItem = screen
        .getAllByRole('menuitem')
        .find((item) => item.textContent === 'Up') as HTMLElement;
      fireEvent.click(upItem.querySelector('input') as HTMLInputElement);

      for (const text of [
        '1 of 2 members up',
        '0 of 0 members up',
        '0 of 1 members up',
      ]) {
        expect([RED, GREEN]).not.toContain(colourOf(screen.getByText(text)));
      }
    });
  });

  describe('opened on ?cluster=<id>', () => {
    const scrollIntoView = vi.fn();

    beforeEach(() => {
      scrollIntoView.mockClear();
      // jsdom has no layout, so no scrollIntoView of its own.
      Element.prototype.scrollIntoView = scrollIntoView;
    });

    afterEach(() => {
      delete (Element.prototype as Partial<Element>).scrollIntoView;
    });

    const estateWithTwoEnvironments = () =>
      topology([
        {
          env_name: 'production',
          clusters: [
            cluster({
              name: 'orders',
              services: [service({ service_name: 'orders-1' })],
            }),
          ],
        },
        {
          env_name: 'staging',
          clusters: [
            cluster({
              name: 'reports',
              services: [service({ service_name: 'reports-1' })],
            }),
            cluster({
              name: 'billing',
              services: [service({ service_name: 'billing-1' })],
            }),
          ],
        },
      ]);

    it('unfolds that cluster, scrolls it into view and moves focus to it', () => {
      const estate = estateWithTwoEnvironments();
      const billing = estate.environments[1].clusters[1];
      useOmTopology.mockReturnValue({
        data: estate,
        isPending: false,
        isError: false,
      });
      renderPage(`/operations?tab=clusters&cluster=${billing.id}`);

      const row = clusterRows().find((candidate) =>
        candidate.textContent?.includes('billing')
      );
      expect(row).toHaveFocus();
      expect(scrollIntoView).toHaveBeenCalledTimes(1);
      expect(scrollIntoView.mock.contexts[0]).toBe(row);
      expect(screen.getByRole('link', { name: 'billing-1' })).toBeVisible();
      expect(screen.queryByRole('link', { name: 'reports-1' })).toBeNull();
      expect(screen.queryByRole('link', { name: 'orders-1' })).toBeNull();
    });

    it('does not scroll again when the topology refetches', () => {
      const estate = estateWithTwoEnvironments();
      const billing = estate.environments[1].clusters[1];
      useOmTopology.mockReturnValue({
        data: estate,
        isPending: false,
        isError: false,
      });
      const { rerender } = renderPage(`/operations?cluster=${billing.id}`);

      useOmTopology.mockReturnValue({
        data: structuredClone(estate),
        isPending: false,
        isError: false,
      });
      rerender(
        <MemoryRouter initialEntries={[`/operations?cluster=${billing.id}`]}>
          <FleetClustersTab />
        </MemoryRouter>
      );

      expect(scrollIntoView).toHaveBeenCalledTimes(1);
    });

    it('does nothing for an id no cluster has', () => {
      renderPage('/operations?tab=clusters&cluster=no-such-cluster');

      expect(document.body).toHaveFocus();
      expect(scrollIntoView).not.toHaveBeenCalled();
      expect(screen.queryByRole('link', { name: 'orders-2' })).toBeNull();
      expect(clusterRows().every((row) => !row.hasAttribute('tabindex'))).toBe(
        true
      );
    });
  });
});
