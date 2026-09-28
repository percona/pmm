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
import { getStoredForm, STORED_FORM_KEY } from './storedForm';

describe('getStoredForm', () => {
  it('returns the stored create-form body when data[_form] is present', () => {
    const task = {
      name: 't',
      data: { [STORED_FORM_KEY]: { task_name: 't', foo: 1 } },
    };
    expect(getStoredForm(task)).toEqual({ task_name: 't', foo: 1 });
  });

  it('returns undefined for a null/undefined task', () => {
    expect(getStoredForm(undefined)).toBeUndefined();
    expect(getStoredForm(null)).toBeUndefined();
  });

  it('returns undefined when data is absent', () => {
    expect(getStoredForm({ name: 't' })).toBeUndefined();
  });

  it('returns undefined for a legacy task whose data has no _form key', () => {
    expect(getStoredForm({ name: 't', data: { meta: {} } })).toBeUndefined();
  });

  it('returns undefined when _form is not an object', () => {
    expect(
      getStoredForm({ name: 't', data: { [STORED_FORM_KEY]: 'nope' } })
    ).toBeUndefined();
    expect(
      getStoredForm({ name: 't', data: { [STORED_FORM_KEY]: null } })
    ).toBeUndefined();
  });

  it('returns undefined when _form is an array', () => {
    expect(
      getStoredForm({ name: 't', data: { [STORED_FORM_KEY]: [] } })
    ).toBeUndefined();
    expect(
      getStoredForm({
        name: 't',
        data: { [STORED_FORM_KEY]: [{ task_name: 't' }] },
      })
    ).toBeUndefined();
  });
});
