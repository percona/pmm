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

import { createContext, useContext } from 'react';
import type { ServiceType } from '../../hooks/useServices';
import type { PluginField } from './types';

const EMPTY_FIELDS: readonly PluginField[] = Object.freeze([]);

/**
 * Flat leaf fields for the active {@link SchemaFormRenderer} form. Used by
 * cascaded widgets (e.g. host ← service) to read sibling field metadata such as
 * ``service_types`` without threading the full section tree through every slot.
 */
const FormFieldsContext = createContext<readonly PluginField[]>(EMPTY_FIELDS);

export const FormFieldsProvider = FormFieldsContext.Provider;

export function useFormFields(): readonly PluginField[] {
  return useContext(FormFieldsContext);
}

/**
 * Return the label of the upstream field named ``dependsOn``, for a cascaded
 * widget that has to tell someone which field to fill in first.
 *
 * A cascade's disabled state is only actionable if it names the control that
 * releases it: "Select a value first" leaves a reader hunting the form for
 * which value, on a form where several fields could plausibly be it. Returns
 * ``undefined`` when there is no parent or it declares no label, so the caller
 * can fall back to the generic wording rather than render an empty phrase.
 */
export function parentLabelForDependsOn(
  fields: readonly PluginField[],
  dependsOn: string | undefined
): string | undefined {
  if (!dependsOn) {
    return undefined;
  }
  const parent = fields.find((field) => field.name === dependsOn);
  return parent?.label || undefined;
}

/**
 * Return the ``service_types`` declared on the upstream service field named
 * ``dependsOn``, if any. Used to bound host-cascade ``useServices`` fetches to
 * the same type filter as the parent selector (and to share its query cache).
 */
export function serviceTypesForDependsOn(
  fields: readonly PluginField[],
  dependsOn: string | undefined
): readonly ServiceType[] | undefined {
  if (!dependsOn) {
    return undefined;
  }
  const parent = fields.find((field) => field.name === dependsOn);
  if (parent?.type === 'service') {
    return parent.service_types as readonly ServiceType[];
  }
  return undefined;
}
