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
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FormProvider, useForm } from 'react-hook-form';
import { MultiChoiceField } from './MultiChoiceField';
import type {
  MultiChoiceField as MultiChoiceFieldType,
  ChoiceOption,
} from '../types';

function Harness({
  field,
  defaultSelected = [],
}: {
  field: MultiChoiceFieldType;
  defaultSelected?: string[];
}) {
  const methods = useForm({ defaultValues: { [field.name]: defaultSelected } });
  return (
    <FormProvider {...methods}>
      <MultiChoiceField field={field} />
    </FormProvider>
  );
}

const CHOICES: ChoiceOption[] = [
  { value: 'a', label: 'Alpha' },
  {
    value: 'b',
    label: 'Beta',
    disabled: true,
    disabled_reason: 'Beta is retired.',
  },
];

describe('MultiChoiceField', () => {
  it('disables the matching option and shows its reason on hover', async () => {
    const user = userEvent.setup();
    const field: MultiChoiceFieldType = {
      type: 'multi_choice',
      name: 'flags',
      label: 'Flags',
      choices: CHOICES,
    };
    render(<Harness field={field} />);

    await user.click(screen.getByRole('combobox'));

    const betaLabel = await screen.findByText('Beta');
    expect(betaLabel.closest('[role="option"]')).toHaveAttribute(
      'aria-disabled',
      'true'
    );

    await user.hover(betaLabel);
    await waitFor(() => {
      expect(screen.getByRole('tooltip')).toHaveTextContent('Beta is retired.');
    });
  });

  it('keeps an already-selected disabled option de-selectable', async () => {
    const user = userEvent.setup();
    const field: MultiChoiceFieldType = {
      type: 'multi_choice',
      name: 'flags',
      label: 'Flags',
      choices: CHOICES,
    };
    render(<Harness field={field} defaultSelected={['b']} />);

    await user.click(screen.getByRole('combobox'));

    // A disabled value that is already selected must stay interactive so the
    // user can clear it; only unselected disabled options are blocked.
    const listbox = await screen.findByRole('listbox');
    const betaOption = within(listbox).getByRole('option', { name: /Beta/ });
    expect(betaOption).not.toHaveAttribute('aria-disabled', 'true');
  });
});
