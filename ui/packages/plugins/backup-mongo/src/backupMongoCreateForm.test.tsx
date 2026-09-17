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
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import type * as SepFramework from '@sep/framework';
import { SuggestedTaskNameEffect } from './backupMongoCreateForm';

/**
 * Stub service resolution with hydrated options only. Scalar rehydration is
 * covered by ``useResolvedServiceField`` tests in ``@sep/framework``.
 */
vi.mock('@sep/framework', async (importOriginal) => {
  const actual = await importOriginal<typeof SepFramework>();
  return {
    ...actual,
    useResolvedServiceField(fieldName: string) {
      const parent = useWatch({ name: fieldName }) as
        | { id: number; name: string }
        | null
        | undefined;
      const hydrated =
        parent && typeof parent === 'object' && typeof parent.name === 'string'
          ? parent
          : undefined;
      return {
        parent,
        service: hydrated,
        resetKey: hydrated ? `id:${hydrated.id}` : 'id:none',
        isResolving: false,
      };
    },
  };
});

const SCHEMA_DEFAULT = 'mongodb-backup';
const FIXED = new Date('2026-07-16T14:30:22.123Z');
const STAMP = '20260716T143022Z';

function TaskNameHarness({
  schemaDefault = SCHEMA_DEFAULT,
}: {
  schemaDefault?: string;
}) {
  const methods = useForm({
    defaultValues: {
      task_name: schemaDefault,
      service_id: null as { id: number; name: string } | null,
    },
  });
  const taskName = methods.watch('task_name');

  return (
    <FormProvider {...methods}>
      <SuggestedTaskNameEffect schemaDefault={schemaDefault} />
      <input
        aria-label="Task name"
        value={taskName}
        onChange={(event) => methods.setValue('task_name', event.target.value)}
      />
      <button
        type="button"
        onClick={() =>
          methods.setValue('service_id', { id: 1, name: 'Alpha Cluster' })
        }
      >
        Pick Alpha
      </button>
      <button
        type="button"
        onClick={() =>
          methods.setValue('service_id', { id: 2, name: 'Beta Cluster' })
        }
      >
        Pick Beta
      </button>
    </FormProvider>
  );
}

describe('SuggestedTaskNameEffect', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(FIXED);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('replaces the schema default with a timestamped suggestion on mount', () => {
    render(<TaskNameHarness />);
    expect(screen.getByLabelText('Task name')).toHaveValue(
      `mongodb-backup-${STAMP}`
    );
  });

  it('updates the suggestion when the service changes while still auto', () => {
    render(<TaskNameHarness />);
    expect(screen.getByLabelText('Task name')).toHaveValue(
      `mongodb-backup-${STAMP}`
    );

    fireEvent.click(screen.getByRole('button', { name: 'Pick Alpha' }));
    expect(screen.getByLabelText('Task name')).toHaveValue(
      `alpha-cluster-backup-${STAMP}`
    );

    fireEvent.click(screen.getByRole('button', { name: 'Pick Beta' }));
    expect(screen.getByLabelText('Task name')).toHaveValue(
      `beta-cluster-backup-${STAMP}`
    );
  });

  it('keeps a manual edit when the service changes', () => {
    render(<TaskNameHarness />);
    expect(screen.getByLabelText('Task name')).toHaveValue(
      `mongodb-backup-${STAMP}`
    );

    fireEvent.change(screen.getByLabelText('Task name'), {
      target: { value: 'my-custom-name' },
    });
    expect(screen.getByLabelText('Task name')).toHaveValue('my-custom-name');

    fireEvent.click(screen.getByRole('button', { name: 'Pick Alpha' }));
    expect(screen.getByLabelText('Task name')).toHaveValue('my-custom-name');

    fireEvent.click(screen.getByRole('button', { name: 'Pick Beta' }));
    expect(screen.getByLabelText('Task name')).toHaveValue('my-custom-name');
  });
});
