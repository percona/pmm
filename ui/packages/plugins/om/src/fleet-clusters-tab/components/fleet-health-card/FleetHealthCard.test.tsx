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
import { ThemeContextProvider, pmmThemeOptions } from '@percona/peak-ui';
import type { MRT_TableInstance } from 'material-react-table';
import { describe, expect, it, vi } from 'vitest';
import type { OmClusterHealth, OmClusterRow } from '../../../types';
import { FleetHealthCard } from './FleetHealthCard';

const rowsOf = (...health: OmClusterHealth[]) =>
  health.map((h, i) => ({ id: String(i), health: h }) as OmClusterRow);

const renderCard = (rows: OmClusterRow[]) => {
  const setFilterValue = vi.fn();
  const table = {
    getColumn: () => ({ setFilterValue }),
  } as unknown as MRT_TableInstance<OmClusterRow>;
  render(
    <ThemeContextProvider themeOptions={pmmThemeOptions}>
      <FleetHealthCard table={table} rows={rows} />
    </ThemeContextProvider>
  );
  return { setFilterValue };
};

describe('FleetHealthCard', () => {
  it('counts the clusters that are not healthy, of all, and each state', () => {
    renderCard(rowsOf('down', 'degraded', 'unknown', 'healthy', 'healthy'));

    expect(screen.getByTestId('om-fleet-health-card')).toHaveTextContent(
      '3/ 5clusters unhealthy'
    );
    expect(screen.getByTestId('om-fleet-health-down')).toHaveTextContent(
      '1 down'
    );
    expect(screen.getByTestId('om-fleet-health-healthy')).toHaveTextContent(
      '2 healthy'
    );
  });

  it('leaves out a state no cluster is in', () => {
    renderCard(rowsOf('down', 'healthy'));

    expect(screen.queryByTestId('om-fleet-health-degraded')).toBeNull();
    expect(screen.queryByTestId('om-fleet-health-unknown')).toBeNull();
  });

  it('filters the table to the unhealthy states present', () => {
    const { setFilterValue } = renderCard(rowsOf('down', 'unknown', 'healthy'));

    fireEvent.click(
      screen.getByRole('button', { name: 'Show unhealthy clusters' })
    );

    expect(setFilterValue).toHaveBeenCalledWith(['down', 'unknown']);
  });

  it('offers no filter when every cluster is healthy', () => {
    renderCard(rowsOf('healthy', 'healthy'));

    expect(screen.getByTestId('om-fleet-health-card')).toHaveTextContent(
      '0/ 2clusters unhealthy'
    );
    expect(
      screen.queryByRole('button', { name: 'Show unhealthy clusters' })
    ).toBeNull();
  });
});
