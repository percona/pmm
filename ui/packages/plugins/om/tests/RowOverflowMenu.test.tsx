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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import MenuItem from '@mui/material/MenuItem';
import { describe, expect, it, vi } from 'vitest';
import { RowOverflowMenu } from '../src/components/RowOverflowMenu';

describe('RowOverflowMenu', () => {
  it('keeps its actions closed until asked', () => {
    render(
      <RowOverflowMenu label="More actions for node00">
        {() => [<MenuItem key="forget">Forget</MenuItem>]}
      </RowOverflowMenu>
    );

    expect(screen.queryByText('Forget')).toBeNull();
  });

  // The destructive action this exists to hide is reached in two steps now rather
  // than one, so the first step has to be findable by name.
  it('opens from a button named for its row', () => {
    render(
      <RowOverflowMenu label="More actions for node00">
        {() => [<MenuItem key="forget">Forget</MenuItem>]}
      </RowOverflowMenu>
    );

    fireEvent.click(
      screen.getByRole('button', { name: 'More actions for node00' })
    );

    expect(screen.getByText('Forget')).toBeInTheDocument();
  });

  it('runs the action and closes behind it', async () => {
    const onForget = vi.fn();
    render(
      <RowOverflowMenu label="More actions for node00">
        {(close) => [
          <MenuItem
            key="forget"
            onClick={() => {
              onForget();
              close();
            }}
          >
            Forget
          </MenuItem>,
        ]}
      </RowOverflowMenu>
    );

    fireEvent.click(screen.getByRole('button', { name: /More actions/ }));
    fireEvent.click(screen.getByText('Forget'));

    expect(onForget).toHaveBeenCalledOnce();
    await waitFor(() => expect(screen.queryByText('Forget')).toBeNull());
  });
});
