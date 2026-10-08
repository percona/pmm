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
import MenuItem from '@mui/material/MenuItem';
import { FormProvider, useForm, type RegisterOptions } from 'react-hook-form';
import { SchemaSelectShell } from './SchemaSelectShell';

const choices = [
  { value: 'apple', label: 'Apple' },
  { value: 'pear', label: 'Pear' },
];

interface ShellOptions {
  value?: string;
  required?: boolean;
  rules?: RegisterOptions;
  tooltip?: string;
  inline?: string;
  /** Dotted for the one-of branch case; see the nested-path test. */
  name?: string;
}

function Harness({
  value = '',
  required,
  rules,
  tooltip,
  inline,
  name = 'fruit',
}: ShellOptions) {
  const methods = useForm({
    defaultValues: { fruit: value, basket: { fruit: value } },
  });
  return (
    <FormProvider {...methods}>
      <form onSubmit={methods.handleSubmit(() => {})}>
        <SchemaSelectShell
          name={name}
          label="Fruit"
          required={required}
          rules={rules}
          tooltip={tooltip}
          inline={inline}
          renderValue={(v) =>
            choices.find((c) => c.value === v)?.label ?? String(v)
          }
        >
          {choices.map((c) => (
            <MenuItem key={c.value} value={c.value}>
              {c.label}
            </MenuItem>
          ))}
        </SchemaSelectShell>
        <button type="submit">Submit</button>
      </form>
    </FormProvider>
  );
}

const renderShell = (opts: ShellOptions = {}) => render(<Harness {...opts} />);

describe('SchemaSelectShell', () => {
  it('keeps the test ids Peak UI derives from the field name', () => {
    renderShell();

    expect(screen.getByTestId('select-fruit-button')).toBeInTheDocument();
    expect(screen.getByTestId('select-input-fruit')).toBeInTheDocument();
    // a11y: the visible combobox is labelled by the InputLabel.
    expect(
      screen.getByRole('combobox').getAttribute('aria-labelledby')
    ).toContain('fruit-input-label');
  });

  // The placeholder and the pinned-notch label were the two select styles this
  // shell used to have that no other control on the form had (PMM-15456).
  it('shows no placeholder and lets the label sit in the empty field', () => {
    const { container } = renderShell();

    expect(screen.queryByText('Select…')).not.toBeInTheDocument();
    expect(container.querySelector('label')).toHaveAttribute(
      'data-shrink',
      'false'
    );
  });

  it('floats the label once a value is chosen, and renders it via renderValue', () => {
    const { container } = renderShell({ value: 'apple' });

    expect(screen.getByText('Apple')).toBeInTheDocument();
    expect(container.querySelector('label')).toHaveAttribute(
      'data-shrink',
      'true'
    );
  });

  it('flips aria-invalid and shows the error message on error', async () => {
    const user = userEvent.setup();
    renderShell({ rules: { required: 'Fruit is required' } });

    await user.click(screen.getByRole('button', { name: 'Submit' }));

    expect(await screen.findByText('Fruit is required')).toBeInTheDocument();
    expect(screen.getByTestId('select-input-fruit')).toHaveAttribute(
      'aria-invalid',
      'true'
    );
  });

  it('shows inline help as helper text when there is no error', () => {
    renderShell({ inline: 'Pick one' });

    expect(screen.getByTestId('select-input-fruit')).toHaveAttribute(
      'aria-invalid',
      'false'
    );
    expect(screen.getByText('Pick one')).toBeInTheDocument();
  });

  it('shows an info-icon tooltip when tooltip help is set', async () => {
    const user = userEvent.setup();
    renderShell({ tooltip: 'Pick one' });

    // MUI's outlined notch clones the label into an aria-hidden <legend>, so
    // the icon exists twice; the visible one is the <label>'s. Same shape the
    // TextInput-backed fields already assert.
    const help = document.querySelector('label [data-help-for="Fruit"]');
    expect(help).toBeInTheDocument();
    await user.hover(help!);
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Pick one');
  });

  it('omits the info icon when description is missing', () => {
    renderShell();

    expect(document.querySelectorAll('[data-help-for="Fruit"]')).toHaveLength(
      0
    );
  });

  it('replaces inline help with the error', async () => {
    const user = userEvent.setup();
    renderShell({
      inline: 'Pick one',
      rules: { required: 'Fruit is required' },
    });

    await user.click(screen.getByRole('button', { name: 'Submit' }));

    expect(await screen.findByText('Fruit is required')).toBeInTheDocument();
    expect(screen.queryByText('Pick one')).not.toBeInTheDocument();
  });

  it('keeps the info icon when an error takes the helper-text slot', async () => {
    const user = userEvent.setup();
    renderShell({
      tooltip: 'Pick one',
      rules: { required: 'Fruit is required' },
    });

    await user.click(screen.getByRole('button', { name: 'Submit' }));

    expect(await screen.findByText('Fruit is required')).toBeInTheDocument();
    expect(
      document.querySelectorAll('label [data-help-for="Fruit"]')
    ).toHaveLength(1);
  });

  // A one-of branch field's name is a dotted path, and react-hook-form nests
  // its error to match. A literal `errors[name]` lookup left the control
  // outlined red with no message under it.
  it('shows the error message for a nested field path', async () => {
    const user = userEvent.setup();
    renderShell({
      name: 'basket.fruit',
      rules: { required: 'Fruit is required' },
    });

    await user.click(screen.getByRole('button', { name: 'Submit' }));

    expect(await screen.findByText('Fruit is required')).toBeInTheDocument();
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
