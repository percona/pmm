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
import { FormProvider, useForm } from 'react-hook-form';
import { BoolField } from './BoolField';
import type { BoolField as BoolFieldType } from '../types';

function Harness({
  field,
  defaultValues,
}: {
  field: BoolFieldType;
  defaultValues?: Record<string, unknown>;
}) {
  const methods = useForm({ defaultValues });
  return (
    <FormProvider {...methods}>
      <BoolField field={field} />
    </FormProvider>
  );
}

describe('BoolField', () => {
  it('renders a switch with label', () => {
    const field: BoolFieldType = {
      type: 'bool',
      name: 'overwrite_tables',
      label: 'Overwrite Tables',
    };
    render(<Harness field={field} />);

    expect(screen.getByLabelText('Overwrite Tables')).toBeInTheDocument();
  });

  it('does not show warning when destructive field is off', () => {
    const field: BoolFieldType = {
      type: 'bool',
      name: 'overwrite_tables',
      label: 'Overwrite Tables',
      destructive: 'This will overwrite existing data on the target database.',
    };
    render(<Harness field={field} defaultValues={{ overwrite_tables: false }} />);

    expect(screen.queryByTestId('destructive-warning')).not.toBeInTheDocument();
  });

  it('shows warning when destructive field is enabled', async () => {
    const user = userEvent.setup();
    const field: BoolFieldType = {
      type: 'bool',
      name: 'overwrite_tables',
      label: 'Overwrite Tables',
      destructive: 'This will overwrite existing data on the target database.',
    };
    render(<Harness field={field} />);

    // Toggle the switch on
    await user.click(screen.getByLabelText('Overwrite Tables'));

    expect(screen.getByTestId('destructive-warning')).toHaveTextContent(
      'This will overwrite existing data on the target database.'
    );
  });

  it('hides warning when destructive field is toggled off', async () => {
    const user = userEvent.setup();
    const field: BoolFieldType = {
      type: 'bool',
      name: 'overwrite_tables',
      label: 'Overwrite Tables',
      destructive: 'This will overwrite existing data on the target database.',
    };
    render(<Harness field={field} defaultValues={{ overwrite_tables: true }} />);

    // Warning should be visible initially
    expect(screen.getByTestId('destructive-warning')).toBeInTheDocument();

    // Toggle the switch off
    await user.click(screen.getByLabelText('Overwrite Tables'));

    expect(screen.queryByTestId('destructive-warning')).not.toBeInTheDocument();
  });

  it('does not show warning for non-destructive fields', () => {
    const field: BoolFieldType = {
      type: 'bool',
      name: 'enable_feature',
      label: 'Enable Feature',
    };
    render(<Harness field={field} defaultValues={{ enable_feature: true }} />);

    expect(screen.queryByTestId('destructive-warning')).not.toBeInTheDocument();
  });
});
