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

import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OmError } from '../src/components/OmError';

describe('OmError', () => {
  it('reads as an alert, framed, with what failed and why', () => {
    render(
      <OmError
        placement="action"
        title="Could not start a scan"
        messages="PMM Extensions did not answer"
      />
    );

    const error = screen.getByRole('alert');
    expect(error).toHaveAttribute('data-placement', 'action');
    expect(error).toHaveTextContent('Could not start a scan');
    expect(error).toHaveTextContent('PMM Extensions did not answer');
    expect(within(error).queryByRole('list')).toBeNull();
  });

  it('lists every failure, never just the first', () => {
    render(
      <OmError
        placement="item"
        messages={['db01: refused', 'db02: timed out', 'db03: gone']}
      />
    );

    expect(
      within(screen.getByRole('list')).getAllByRole('listitem')
    ).toHaveLength(3);
  });

  it('drops empty entries rather than drawing blank lines', () => {
    render(
      <OmError
        placement="load"
        messages={['', 'the reason', null, undefined]}
      />
    );

    expect(screen.getByRole('alert')).toHaveTextContent(/^the reason$/);
    expect(screen.queryByRole('list')).toBeNull();
  });
});
