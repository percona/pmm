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
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { GeneratedPasswordDialog } from '../src/components/GeneratedPasswordDialog';

const renderDialog = () =>
  render(
    <GeneratedPasswordDialog
      username="admin"
      password="generated-secret"
      onSaved={vi.fn()}
    />
  );

describe('GeneratedPasswordDialog', () => {
  // The shared clipboard hook only writes from a secure context, which jsdom is not.
  beforeEach(() => {
    vi.stubGlobal('isSecureContext', true);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('keeps the password hidden until the eye is pressed', async () => {
    renderDialog();
    const field = screen.getByLabelText('Password');
    expect(field).toHaveAttribute('type', 'password');

    await userEvent.click(
      screen.getByRole('button', { name: 'Show password' })
    );
    expect(field).toHaveAttribute('type', 'text');
    expect(field).toHaveValue('generated-secret');

    await userEvent.click(
      screen.getByRole('button', { name: 'Hide password' })
    );
    expect(field).toHaveAttribute('type', 'password');
  });

  it('copies the password without showing it', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    });
    renderDialog();

    await userEvent.click(
      screen.getByRole('button', { name: 'Copy password' })
    );

    expect(writeText).toHaveBeenCalledWith('generated-secret');
    expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument();
    expect(screen.getByLabelText('Password')).toHaveAttribute(
      'type',
      'password'
    );
  });

  it('says how to copy by hand when the clipboard is out of reach', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: vi.fn().mockRejectedValue(new Error('denied')) },
      configurable: true,
    });
    renderDialog();

    await userEvent.click(
      screen.getByRole('button', { name: 'Copy password' })
    );

    expect(
      screen.getByText(
        'Copying is not available here. Show the password and copy it by hand.'
      )
    ).toBeInTheDocument();
  });
});
