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

import { useWatch } from 'react-hook-form';
import {
  serviceTypesForDependsOn,
  useFormFields,
} from '../components/SchemaFormRenderer/formFieldsContext';
import { cascadeParentResetKey } from '../utils/cascadeParentResetKey';
import { extractId } from '../utils/extractId';
import {
  useServices,
  type ServiceOption,
  type ServiceType,
} from './useServices';

const EMPTY_SERVICES: ServiceOption[] = [];

function isHydratedService(value: unknown): value is ServiceOption {
  return Boolean(value && typeof value === 'object' && 'name' in value);
}

export interface ResolvedServiceField {
  /** Raw watched form value (object, scalar id, or empty). */
  parent: unknown;
  /** Hydrated service when available; undefined while resolving or when empty. */
  service: ServiceOption | undefined;
  /** Stable key for cascading resets when the parent selection changes. */
  resetKey: string;
  /**
   * True while a scalar id is awaiting a bounded ``useServices`` fetch.
   * Callers that need the display name should wait rather than treating the
   * parent as missing.
   */
  isResolving: boolean;
  /**
   * True when the bounded ``useServices`` fetch failed. ``service`` stays
   * undefined and ``isResolving`` is false, so callers need this to tell a
   * failed lookup apart from an id that matched no service.
   */
  isError: boolean;
  /** Error from the bounded ``useServices`` fetch, when it failed. */
  error: Error | null;
}

/**
 * Watch a service form field and resolve scalar ids to a hydrated
 * {@link ServiceOption} the same way host cascade does.
 *
 * When ``serviceTypes`` is omitted, types are read from the parent field via
 * {@link FormFieldsProvider} (``service_types`` on the named service field).
 * Fetch stays disabled until a non-empty type filter is available
 * (``undefined`` or ``[]`` both keep it off) so we never page the untyped
 * services list that ``useServices`` would otherwise treat as "all types".
 */
export function useResolvedServiceField(
  fieldName: string,
  serviceTypes?: readonly ServiceType[]
): ResolvedServiceField {
  const fields = useFormFields();
  const types = serviceTypes ?? serviceTypesForDependsOn(fields, fieldName);
  const parent = useWatch({ name: fieldName }) as
    | ServiceOption
    | number
    | string
    | null
    | undefined;

  const unresolvedServiceId = isHydratedService(parent)
    ? null
    : extractId(parent);
  // Require length > 0: ``[]`` normalises in useServices to an unbound fetch.
  const enabled = unresolvedServiceId !== null && Boolean(types?.length);
  const {
    data: services = EMPTY_SERVICES,
    isFetched,
    isError,
    error,
  } = useServices({
    serviceTypes: types,
    enabled,
  });

  const service: ServiceOption | undefined = isHydratedService(parent)
    ? parent
    : unresolvedServiceId !== null
      ? services.find((svc) => svc.id === unresolvedServiceId)
      : undefined;

  return {
    parent,
    service,
    resetKey: cascadeParentResetKey(parent),
    isResolving: enabled && !isFetched,
    isError: enabled && isError,
    error: enabled ? (error ?? null) : null,
  };
}
