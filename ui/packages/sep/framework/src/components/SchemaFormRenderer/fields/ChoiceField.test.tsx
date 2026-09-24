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
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FormProvider, useForm } from 'react-hook-form';
import { ChoiceField } from './ChoiceField';
import type { ChoiceField as ChoiceFieldType, ChoiceOption } from '../types';

function Harness({ field }: { field: ChoiceFieldType }) {
  const methods = useForm();
  return (
    <FormProvider {...methods}>
      <ChoiceField field={field} />
    </FormProvider>
  );
}

function SubmitHarness({ field }: { field: ChoiceFieldType }) {
  const methods = useForm();
  return (
    <FormProvider {...methods}>
      <form onSubmit={methods.handleSubmit(() => {})}>
        <ChoiceField field={field} />
        <button type="submit">Submit</button>
      </form>
    </FormProvider>
  );
}

function renderField(
  choices: ChoiceOption[],
  overrides: Partial<ChoiceFieldType> = {}
) {
  const field: ChoiceFieldType = {
    type: 'choice',
    name: 'archive_type',
    label: 'Archive Type',
    choices,
    ...overrides,
  };
  return render(<Harness field={field} />);
}

// Three or fewer options render as radios; more render as a select.
const RADIO_CHOICES: ChoiceOption[] = [
  { value: '0', label: 'Purge Only' },
  {
    value: '1',
    label: 'Swap & Drop',
    disabled: true,
    disabled_reason: 'Not available in the current scope.',
  },
];

const SELECT_CHOICES: ChoiceOption[] = [
  { value: 'a', label: 'Alpha' },
  { value: 'b', label: 'Beta' },
  { value: 'c', label: 'Gamma' },
  {
    value: 'd',
    label: 'Delta',
    disabled: true,
    disabled_reason: 'Delta is retired.',
  },
];

describe('ChoiceField — radio branch', () => {
  it('disables the option and surfaces its reason via a tooltip on hover', async () => {
    const user = userEvent.setup();
    renderField(RADIO_CHOICES);

    const radioByValue = (value: string) =>
      screen
        .getAllByRole('radio')
        .find((radio) => (radio as HTMLInputElement).value === value);
    expect(radioByValue('1')).toBeDisabled();
    expect(radioByValue('0')).not.toBeDisabled();

    await user.hover(screen.getByText('Swap & Drop'));
    await waitFor(() => {
      expect(screen.getByRole('tooltip')).toHaveTextContent(
        'Not available in the current scope.'
      );
    });
  });

  it('does not render a tooltip for an enabled option', async () => {
    const user = userEvent.setup();
    renderField(RADIO_CHOICES);

    await user.hover(screen.getByText('Purge Only'));
    // Give any (unexpected) tooltip a chance to appear before asserting absence.
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });

  it('surfaces the required-validation error on submit', async () => {
    const user = userEvent.setup();
    const field: ChoiceFieldType = {
      type: 'choice',
      name: 'archive_type',
      label: 'Archive Type',
      choices: RADIO_CHOICES,
      required: true,
    };
    render(<SubmitHarness field={field} />);

    await user.click(screen.getByRole('button', { name: 'Submit' }));

    await waitFor(() => {
      expect(screen.getByText('Archive Type is required')).toBeInTheDocument();
    });
  });
});

describe('ChoiceField — select branch', () => {
  it('disables the matching MenuItem and shows its reason on hover', async () => {
    const user = userEvent.setup();
    renderField(SELECT_CHOICES);

    await user.click(screen.getByRole('combobox'));

    const deltaLabel = await screen.findByText('Delta');
    expect(deltaLabel.closest('[role="option"]')).toHaveAttribute(
      'aria-disabled',
      'true'
    );

    await user.hover(deltaLabel);
    await waitFor(() => {
      expect(screen.getByRole('tooltip')).toHaveTextContent(
        'Delta is retired.'
      );
    });
  });
});
