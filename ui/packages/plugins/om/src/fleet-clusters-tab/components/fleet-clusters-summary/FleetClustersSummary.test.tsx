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
import { ThemeContextProvider, pmmThemeOptions } from '@percona/peak-ui';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  cluster,
  mixedEstate,
  service,
  topology,
} from '../../../../tests/fixtures';
import { FleetClustersSummary } from './FleetClustersSummary';

const { useOmTopology } = vi.hoisted(() => ({ useOmTopology: vi.fn() }));

vi.mock('../../../topologyHooks', () => ({ useOmTopology }));

const renderSummary = () =>
  render(
    <ThemeContextProvider themeOptions={pmmThemeOptions}>
      <FleetClustersSummary fallback="The fleet's clusters." />
    </ThemeContextProvider>
  );

describe('FleetClustersSummary', () => {
  beforeEach(() => {
    useOmTopology.mockReturnValue({ data: mixedEstate() });
  });

  it('counts the clusters and each state, worst first, leaving out the empty ones', () => {
    useOmTopology.mockReturnValue({
      data: topology([
        {
          env_name: 'production',
          clusters: [
            cluster({ name: 'a', services: [service({})] }),
            cluster({ name: 'b', services: [service({})] }),
            cluster({
              name: 'c',
              services: [service({ status: 'SERVICE_STATUS_DOWN' })],
            }),
          ],
        },
      ]),
    });

    renderSummary();

    expect(screen.getByTestId('om-fleet-summary')).toHaveTextContent(
      '3 clusters1 down2 healthy'
    );
    expect(screen.queryByText(/degraded|unknown/)).toBeNull();
    expect(
      screen.getByTestId('om-cluster-health-icon-down')
    ).toBeInTheDocument();
  });

  it('counts the whole fleet', () => {
    renderSummary();

    expect(screen.getByText('2 clusters')).toBeInTheDocument();
  });

  it('keeps the fallback until there is a cluster to count', () => {
    useOmTopology.mockReturnValue({ data: undefined });

    renderSummary();

    expect(screen.getByText("The fleet's clusters.")).toBeInTheDocument();
    expect(screen.queryByTestId('om-fleet-summary')).toBeNull();
  });
});
