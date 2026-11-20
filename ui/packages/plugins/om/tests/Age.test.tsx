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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { browserTimezone } from '@pmm-extensions/framework';
import { Age } from '../src/components/Age';

const NOW = '2026-10-06T12:00:00Z';

describe('Age', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date(NOW));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('keeps the compact age and puts the local time and zone on hover', async () => {
    const iso = '2026-10-06T10:48:00Z';
    render(<Age value={iso} />);

    const age = screen.getByText('1h 12m ago');
    fireEvent.mouseOver(age);
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      `${new Date(iso).toLocaleString()} (${browserTimezone()})`
    );
    // The time describes the age; it does not replace it as the name.
    expect(age).not.toHaveAttribute('aria-label');
    // MUI opens the hover on focus too, so a keyboard reaches it.
    expect(age).toHaveAttribute('tabindex', '0');
  });

  it('says what the time is of, when given a label', async () => {
    const iso = '2026-10-06T10:48:00Z';
    render(<Age value={iso} label="Last collected" />);

    fireEvent.mouseOver(screen.getByText('1h 12m ago'));
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      `Last collected ${new Date(iso).toLocaleString()} (${browserTimezone()})`
    );
  });

  it('renders an em-dash with no hover when there is no timestamp', () => {
    render(<Age value={null} />);

    expect(screen.getByText('—')).not.toHaveAttribute('title');
    expect(screen.getByText('—')).not.toHaveAttribute('tabindex');
  });
});
