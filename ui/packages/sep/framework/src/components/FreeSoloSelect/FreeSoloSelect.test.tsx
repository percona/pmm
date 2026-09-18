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

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { FormProvider, useForm } from 'react-hook-form';
import { FreeSoloSelect } from './FreeSoloSelect';
import type { ReferenceOption } from './freeSoloValue';

const OPTIONS: ReferenceOption[] = [
  { id: 10, name: 'app_prod' },
  { id: 11, name: 'analytics' },
];

const labelOf = (o: ReferenceOption) => o.name;

function Harness({
  defaultValue = '',
  options = OPTIONS,
}: {
  defaultValue?: unknown;
  options?: readonly ReferenceOption[];
}) {
  const methods = useForm({ defaultValues: { schema: defaultValue } });
  return (
    <FormProvider {...methods}>
      <FreeSoloSelect<ReferenceOption>
        name="schema"
        label="Schema"
        options={options}
        getOptionLabel={labelOf}
      />
      <output data-testid="value">
        {JSON.stringify(methods.watch('schema'))}
      </output>
    </FormProvider>
  );
}

const value = () => screen.getByTestId('value').textContent;

describe('FreeSoloSelect', () => {
  it('commits the inventory id when an option is picked', async () => {
    const user = userEvent.setup();
    render(<Harness />);
    await user.click(screen.getByLabelText('Schema'));
    await user.click(await screen.findByText('app_prod'));
    expect(value()).toBe('10');
  });

  it('commits a string when a novel value is typed', async () => {
    const user = userEvent.setup();
    render(<Harness />);
    await user.type(screen.getByLabelText('Schema'), 'custom_db');
    expect(value()).toBe('"custom_db"');
  });

  it('resolves a typed value matching an existing option to its id', async () => {
    const user = userEvent.setup();
    render(<Harness />);
    await user.type(screen.getByLabelText('Schema'), 'analytics');
    expect(value()).toBe('11');
  });

  it('yields an empty value when cleared', async () => {
    const user = userEvent.setup();
    render(<Harness defaultValue={10} />);
    await user.click(screen.getByLabelText('Clear'));
    expect(value()).toBe('null');
  });

  it('renders a typed (non-inventory) value distinctly in the dropdown', async () => {
    const user = userEvent.setup();
    render(<Harness />);
    await user.type(screen.getByLabelText('Schema'), 'brand_new');
    const suggestion = await screen.findByText('brand_new');
    expect(suggestion.tagName).toBe('EM');
  });

  it('round-trips a prefilled inventory id to its option label', () => {
    render(<Harness defaultValue={11} />);
    expect(screen.getByLabelText('Schema')).toHaveValue('analytics');
  });

  it('re-resolves a value typed before options loaded to its id once they arrive', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<Harness options={[]} />);
    // Type an exact label while options are still empty → commits as a string.
    await user.type(screen.getByLabelText('Schema'), 'analytics');
    expect(value()).toBe('"analytics"');
    // Options arrive → the stored string resolves to the matching id.
    rerender(<Harness options={OPTIONS} />);
    await waitFor(() => expect(value()).toBe('11'));
  });
});
