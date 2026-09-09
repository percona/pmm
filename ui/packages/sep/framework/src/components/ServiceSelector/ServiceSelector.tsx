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

import { useEffect } from 'react';
import { get, useFormContext } from 'react-hook-form';
import { AutoCompleteInput } from '@percona/peak-ui';
import {
  useServices,
  type ServiceOption,
  type ServiceType,
} from '../../hooks/useServices';
import { extractId } from '../../utils/extractId';
import {
  isHydratedReferenceOption,
  resolveReferenceOption,
} from '../../utils/referenceOption';
import { FreeSoloSelect } from '../FreeSoloSelect';

const EMPTY_OPTIONS: ServiceOption[] = [];

export interface ServiceSelectorProps {
  /**
   * react-hook-form field name. Without `allowCustom`, stores a
   * `ServiceOption | null`; with `allowCustom`, stores the committed
   * `number | string | null` (inventory id, free-typed value, or unset).
   */
  name: string;
  label: string;
  required?: boolean;
  /** Optional filter — only services whose `type` is in this list are shown. */
  serviceTypes?: readonly ServiceType[];
  disabled?: boolean;
  helperText?: string;
  /** Offer free-text (free-solo) entry alongside the inventory options. */
  allowCustom?: boolean;
}

const getOptionLabel = (opt: ServiceOption | string) =>
  typeof opt === 'string' ? opt : `${opt.name} (${opt.type})`;
// Used by the free-solo path. Because the label is `name (type)`, the
// typed-text-to-id resolution only fires if the user types that exact form;
// in practice a free-typed service almost always commits as a custom string,
// while inventory picks resolve to an id through the option-click path.
const getServiceOptionLabel = (opt: ServiceOption) =>
  `${opt.name} (${opt.type})`;

const isOptionEqualToValue = (a: ServiceOption, b: ServiceOption) =>
  a.id === b.id;

export function ServiceSelector({
  name,
  label,
  required,
  serviceTypes,
  disabled,
  helperText,
  allowCustom,
}: ServiceSelectorProps) {
  const {
    control,
    setValue,
    watch,
    formState: { errors },
  } = useFormContext();
  const storedValue = watch(name);
  const fieldError = get(errors, name)?.message as string | undefined;

  const {
    data: services = EMPTY_OPTIONS,
    isLoading,
    isError,
    error,
  } = useServices({ serviceTypes });

  const empty = !isLoading && !isError && services.length === 0;

  const showError = isError || Boolean(fieldError);
  const text = fieldError
    ? fieldError
    : isError
      ? (error?.message ?? 'Failed to load services')
      : empty
        ? 'No services available'
        : helperText;

  useEffect(() => {
    if (allowCustom || isHydratedReferenceOption(storedValue)) {
      return;
    }
    const id = extractId(storedValue);
    if (id === null) {
      return;
    }
    const match = services.find((service) => service.id === id);
    if (match) {
      setValue(name, match, { shouldDirty: false, shouldValidate: false });
    }
  }, [allowCustom, storedValue, services, name, setValue]);

  if (allowCustom) {
    return (
      <FreeSoloSelect<ServiceOption>
        name={name}
        label={label}
        options={services}
        getOptionLabel={getServiceOptionLabel}
        required={required}
        disabled={disabled || isError}
        loading={isLoading}
        helperText={text}
        error={showError}
        noOptionsText={
          isLoading ? 'Loading services…' : 'No services available'
        }
      />
    );
  }

  return (
    <AutoCompleteInput<ServiceOption>
      name={name}
      label={label}
      control={control}
      isRequired={required}
      loading={isLoading}
      disabled={disabled || isError}
      options={services}
      controllerProps={{
        rules: required ? { required: `${label} is required` } : undefined,
      }}
      autoCompleteProps={{
        getOptionLabel,
        isOptionEqualToValue,
        noOptionsText: isLoading
          ? 'Loading services…'
          : 'No services available',
        value: resolveReferenceOption(storedValue, services),
      }}
      textFieldProps={{ helperText: text, error: showError }}
    />
  );
}
