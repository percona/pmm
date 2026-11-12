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
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { OM_ROUTE_INSTALL } from '../src/constants';
import { OmApp } from '../src/OmApp';
import { asReadOnlyReader } from './auth';

vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryRuns: () => ({ data: [] }),
}));
vi.mock('../src/BootstrapPage', () => ({
  BootstrapPage: () => <div data-testid="bootstrap-page" />,
}));

const MOUNT = '/operations';

const openInstall = () =>
  render(
    <MemoryRouter initialEntries={[`${MOUNT}/${OM_ROUTE_INSTALL}`]}>
      <Routes>
        <Route path={`${MOUNT}/*`} element={<OmApp />} />
      </Routes>
    </MemoryRouter>
  );

describe('OmApp', () => {
  it('serves the install page to a reader who may act', () => {
    openInstall();

    expect(screen.getByTestId('bootstrap-page')).toBeInTheDocument();
  });

  it('answers a direct link to the install page with the Admin role it needs', () => {
    asReadOnlyReader();
    openInstall();

    expect(screen.queryByTestId('bootstrap-page')).toBeNull();
    expect(screen.getByTestId('read-only-notice')).toHaveTextContent(
      "You don't have permission to install MongoDB."
    );
  });
});
