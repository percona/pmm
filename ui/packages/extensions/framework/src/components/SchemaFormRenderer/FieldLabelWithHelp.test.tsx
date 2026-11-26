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

import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FieldHelpIcon, FieldLabelWithHelp } from './FieldLabelWithHelp';

describe('FieldLabelWithHelp', () => {
  it('returns the plain label string when description is missing', () => {
    const { container } = render(<FieldLabelWithHelp label="Title" />);

    expect(screen.getByText('Title')).toBeInTheDocument();
    expect(screen.queryByLabelText('Help for Title')).not.toBeInTheDocument();
    // Plain string path — no wrapper span around the label alone.
    expect(container.querySelector('span')).toBeNull();
  });

  it('renders an info icon and tooltip when description is set', async () => {
    const user = userEvent.setup();
    render(<FieldLabelWithHelp label="Title" description="A title" />);

    expect(screen.getByText('Title')).toBeInTheDocument();
    const help = screen.getByLabelText('Help for Title');
    expect(help).toHaveAttribute('data-help-for', 'Title');

    await user.hover(help);
    expect(await screen.findByRole('tooltip')).toHaveTextContent('A title');
  });
});

describe('FieldHelpIcon', () => {
  it('exposes the description via tooltip on hover', async () => {
    const user = userEvent.setup();
    render(<FieldHelpIcon label="Enabled" description="Turn this on" />);

    await user.hover(screen.getByLabelText('Help for Enabled'));
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Turn this on'
    );
  });

  it('is keyboard-focusable and describes the child with the tooltip title', () => {
    render(<FieldHelpIcon label="Enabled" description="Turn this on" />);

    const help = screen.getByLabelText('Help for Enabled');
    expect(help).toHaveAttribute('tabindex', '0');
    expect(help).toHaveAttribute('data-help-for', 'Enabled');
    // describeChild keeps aria-label as the name; description via title /
    // aria-describedby.
    expect(help).toHaveAccessibleDescription('Turn this on');
  });
});
