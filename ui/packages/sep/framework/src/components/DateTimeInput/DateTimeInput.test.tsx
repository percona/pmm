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
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FormProvider, useForm } from 'react-hook-form';
import { DateTimeInput } from './DateTimeInput';

function Harness({
  defaultValue = '',
  isRequired,
  onSubmit = vi.fn(),
}: {
  defaultValue?: string;
  isRequired?: boolean;
  onSubmit?: (values: { at: string }) => void;
}) {
  const methods = useForm({ defaultValues: { at: defaultValue } });
  return (
    <FormProvider {...methods}>
      <form onSubmit={methods.handleSubmit(onSubmit)}>
        <DateTimeInput
          name="at"
          control={methods.control}
          label="Start time (UTC)"
          isRequired={isRequired}
          controllerProps={
            isRequired ? { rules: { required: 'Pick a time' } } : undefined
          }
        />
        <button type="submit">Save</button>
      </form>
    </FormProvider>
  );
}

const input = () =>
  screen.getByTestId('date-time-picker-at') as HTMLInputElement;

describe('DateTimeInput', () => {
  // The control mounts its own LocalizationProvider; without one the picker
  // throws for want of a date adapter, so rendering at all is the assertion.
  it('renders without the host supplying a date adapter', () => {
    render(<Harness defaultValue="2026-03-01T14:30" />);

    expect(input().value).toBe('03/01/2026 02:30 PM');
  });

  it('writes the picker value back as a wall-clock string', async () => {
    const onSubmit = vi.fn();
    const user = userEvent.setup();
    render(<Harness defaultValue="2026-03-01T14:30" onSubmit={onSubmit} />);

    fireEvent.change(input(), { target: { value: '04/05/2027 09:15 AM' } });
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ at: '2027-04-05T09:15' }),
      expect.anything()
    );
  });

  it('keeps an empty field as an empty string, not null', async () => {
    const onSubmit = vi.fn();
    const user = userEvent.setup();
    render(<Harness defaultValue="2026-03-01T14:30" onSubmit={onSubmit} />);

    await user.clear(input());
    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ at: '' }),
      expect.anything()
    );
  });

  // The one case the local-time bridge cannot render is a wall clock inside
  // the reader's own spring-forward gap, which `Date` normalizes forward (see
  // `wallClockValue.ts`). That stays display-only as long as mounting does not
  // write the normalized value back into the form — so pin that.
  it('does not touch the form value or dirty the field on mount', async () => {
    const onSubmit = vi.fn();
    const user = userEvent.setup();
    render(<Harness defaultValue="2026-03-08T02:30" onSubmit={onSubmit} />);

    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ at: '2026-03-08T02:30' }),
      expect.anything()
    );
  });

  it('marks the label required and blocks submission on a required rule', async () => {
    const onSubmit = vi.fn();
    const user = userEvent.setup();
    const { container } = render(<Harness isRequired onSubmit={onSubmit} />);

    expect(
      container.querySelector('.MuiFormLabel-asterisk')
    ).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Save' }));

    expect(onSubmit).not.toHaveBeenCalled();
  });
});
