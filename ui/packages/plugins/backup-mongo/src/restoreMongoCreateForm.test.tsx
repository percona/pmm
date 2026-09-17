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
import type { ReactElement } from 'react';
import {
  restoreMongoCreateForm,
  restoreMongoEditForm,
} from './restoreMongoCreateForm';

function slotProps(overrides: Record<string, unknown> = {}) {
  return {
    sections: [],
    onSubmit: vi.fn(),
    loading: false,
    capabilities: undefined,
    renderField: undefined,
    ...overrides,
  } as Parameters<typeof restoreMongoCreateForm>[0];
}

// Both slots return <div><SchemaFormRenderer .../></div>; read the renderer props
// without a DOM render (SchemaFormRenderer is not invoked, only referenced).
function rendererProps(
  node: ReturnType<typeof restoreMongoCreateForm>
): Record<string, unknown> {
  const wrapper = node as ReactElement<{ children: ReactElement }>;
  return wrapper.props.children.props as Record<string, unknown>;
}

describe('restoreMongoEditForm', () => {
  it('renders the Save label and defers to the edit page stored-form defaults', () => {
    const defaultValues = {
      task_name: 'existing-restore',
      hostname: 'mongo-host',
    };

    const props = rendererProps(
      restoreMongoEditForm(slotProps({ defaultValues }))
    );

    expect(props.submitLabel).toBe('Save');
    expect(props.defaultValues).toEqual(defaultValues);
  });

  it('does not inject the create-only task-name suggestion', () => {
    const props = rendererProps(
      restoreMongoEditForm(slotProps({ defaultValues: {} }))
    );

    expect(props.defaultValues).toEqual({});
  });

  it('forwards submit and field errors so a 422 renders inline, not just a snackbar', () => {
    const fieldErrors = [{ path: 'hostname', message: 'Unknown host' }];

    const props = rendererProps(
      restoreMongoEditForm(
        slotProps({ submitError: 'Validation failed', fieldErrors })
      )
    );

    expect(props.submitError).toBe('Validation failed');
    expect(props.fieldErrors).toEqual(fieldErrors);
  });
});

describe('restoreMongoCreateForm', () => {
  it('renders the create label and a suggested task name', () => {
    const props = rendererProps(restoreMongoCreateForm(slotProps()));

    expect(props.submitLabel).toBe('Create MongoDB Restores');
    expect((props.defaultValues as { task_name: string }).task_name).toMatch(
      /^mongodb-restore/
    );
  });
});
