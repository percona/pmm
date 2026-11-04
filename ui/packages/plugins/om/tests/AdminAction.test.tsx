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

import { Button } from '@mui/material';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { ADMIN_REQUIRED, AdminAction } from '../src/components/AdminAction';
import { asReadOnlyReader } from './auth';

const renderWithHint = () =>
  render(
    <>
      <span id="hint">Runs on every node</span>
      <AdminAction>
        <Button aria-describedby="hint">Scan all</Button>
      </AdminAction>
    </>
  );

describe('AdminAction', () => {
  it('leaves the action and its description alone for an admin', () => {
    renderWithHint();

    const button = screen.getByRole('button', { name: 'Scan all' });
    expect(button).toBeEnabled();
    expect(button).toHaveAccessibleDescription('Runs on every node');
  });

  it("adds the reason to the action's own description for a reader", () => {
    asReadOnlyReader();
    renderWithHint();

    const button = screen.getByRole('button', { name: 'Scan all' });
    expect(button).toBeDisabled();
    expect(button).toHaveAccessibleDescription(
      `Runs on every node ${ADMIN_REQUIRED}`
    );
  });

  it('lets a keyboard user reach the reason a disabled action gives', async () => {
    asReadOnlyReader();
    renderWithHint();

    await userEvent.tab();

    const wrapper = screen.getByRole('button', { name: 'Scan all' })
      .parentElement as HTMLElement;
    expect(wrapper).toHaveFocus();
    expect(wrapper).toHaveAccessibleName(ADMIN_REQUIRED);
  });
});
