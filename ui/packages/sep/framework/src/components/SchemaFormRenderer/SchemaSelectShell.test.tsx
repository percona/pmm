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

import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import MenuItem from '@mui/material/MenuItem';
import type {
  ControllerRenderProps,
  FieldError,
  FieldValues,
} from 'react-hook-form';
import { SchemaSelectShell } from './SchemaSelectShell';

const choices = [
  { value: 'apple', label: 'Apple' },
  { value: 'pear', label: 'Pear' },
];

function makeField(
  overrides: Partial<ControllerRenderProps> = {}
): ControllerRenderProps<FieldValues, string> {
  return {
    name: 'fruit',
    value: '',
    onChange: vi.fn(),
    onBlur: vi.fn(),
    ref: vi.fn(),
    ...overrides,
  } as ControllerRenderProps<FieldValues, string>;
}

function renderShell(
  opts: {
    field?: Partial<ControllerRenderProps>;
    required?: boolean;
    error?: FieldError;
    description?: string;
  } = {}
) {
  return render(
    <SchemaSelectShell
      field={makeField(opts.field)}
      labelId="fruit-label"
      label="Fruit"
      required={opts.required}
      error={opts.error}
      description={opts.description}
      renderValue={(value) =>
        value === undefined || value === null || value === ''
          ? 'Select…'
          : (choices.find((c) => c.value === value)?.label ?? String(value))
      }
    >
      {choices.map((c) => (
        <MenuItem key={c.value} value={c.value}>
          {c.label}
        </MenuItem>
      ))}
    </SchemaSelectShell>
  );
}

describe('SchemaSelectShell', () => {
  it('renders the placeholder branch and preserves the test ids when empty', () => {
    renderShell();

    expect(screen.getByText('Select…')).toBeInTheDocument();
    expect(screen.getByTestId('select-fruit-button')).toBeInTheDocument();
    expect(screen.getByTestId('select-input-fruit')).toBeInTheDocument();
    // a11y: the visible combobox is labelled by the InputLabel.
    expect(
      screen.getByRole('combobox').getAttribute('aria-labelledby')
    ).toContain('fruit-label');
  });

  it('renders the populated value via renderValue', () => {
    renderShell({ field: { value: 'apple' } });

    expect(screen.getByText('Apple')).toBeInTheDocument();
    expect(screen.queryByText('Select…')).not.toBeInTheDocument();
  });

  it('flips aria-invalid and shows the error message on error', () => {
    renderShell({ error: { type: 'required', message: 'Fruit is required' } });

    expect(screen.getByTestId('select-input-fruit')).toHaveAttribute(
      'aria-invalid',
      'true'
    );
    expect(screen.getByText('Fruit is required')).toBeInTheDocument();
  });

  it('shows the description as helper text when there is no error', () => {
    renderShell({ description: 'Pick one' });

    expect(screen.getByTestId('select-input-fruit')).toHaveAttribute(
      'aria-invalid',
      'false'
    );
    expect(screen.getByText('Pick one')).toBeInTheDocument();
  });

  it('renders a required asterisk in the label', () => {
    const { container } = renderShell({ required: true });

    expect(
      container.querySelector('.MuiFormLabel-asterisk')
    ).toBeInTheDocument();
  });

  it('omits the required asterisk when not required', () => {
    const { container } = renderShell();

    expect(
      container.querySelector('.MuiFormLabel-asterisk')
    ).not.toBeInTheDocument();
  });
});
