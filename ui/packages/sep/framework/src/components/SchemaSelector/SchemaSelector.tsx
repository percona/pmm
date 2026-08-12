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

import { useEffect, useRef } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import { AutoCompleteInput } from '@percona/percona-ui';
import { useSchemas, type SchemaOption } from '../../hooks/useSchemas';
import type { ServiceOption } from '../../hooks/useServices';
import { extractId } from '../../utils/extractId';
import { FreeSoloSelect } from '../FreeSoloSelect';

const EMPTY_OPTIONS: SchemaOption[] = [];

export interface SchemaSelectorProps {
  /**
   * react-hook-form field name. Without `allowCustom`, stores a
   * `SchemaOption | null`; with `allowCustom`, stores the committed
   * `number | string | null` (inventory id, free-typed value, or unset).
   */
  name: string;
  label: string;
  required?: boolean;
  /**
   * Form field name of the parent `ServiceSelector`. The watched value is
   * either a `ServiceOption` (from `<ServiceSelector>`) or a raw service id.
   */
  dependsOn: string;
  disabled?: boolean;
  /** Offer free-text (free-solo) entry alongside the inventory options. */
  allowCustom?: boolean;
}

const getOptionLabel = (opt: SchemaOption | string) =>
  typeof opt === 'string' ? opt : opt.name;
const getOptionName = (opt: SchemaOption) => opt.name;

const isOptionEqualToValue = (a: SchemaOption, b: SchemaOption) =>
  a.id === b.id;

export function SchemaSelector({
  name,
  label,
  required,
  dependsOn,
  disabled,
  allowCustom,
}: SchemaSelectorProps) {
  const { control, setValue } = useFormContext();

  const parent = useWatch({ control, name: dependsOn }) as
    | ServiceOption
    | number
    | string
    | null
    | undefined;
  const parentText = typeof parent === 'string' ? parent.trim() : '';
  const parentIsCustom = Boolean(allowCustom) && parentText !== '';
  const serviceId = parentIsCustom ? null : extractId(parent);
  // A parent that resolves to no inventory id but holds a non-empty string is a
  // free-typed (custom) parent value, not an absent one. A free-solo child can
  // still accept a typed value in that case (it just has no options to offer).
  const parentResetKey = parentIsCustom
    ? `custom:${parentText}`
    : `id:${serviceId ?? 'none'}`;

  const prevParentKeyRef = useRef(parentResetKey);
  useEffect(() => {
    if (prevParentKeyRef.current !== parentResetKey) {
      prevParentKeyRef.current = parentResetKey;
      setValue(name, null, { shouldDirty: true, shouldValidate: false });
    }
  }, [parentResetKey, name, setValue]);

  const {
    data: schemas = EMPTY_OPTIONS,
    isLoading,
    isError,
    error,
  } = useSchemas({ serviceId });

  const noService = serviceId === null || serviceId === undefined;
  const empty = !noService && !isLoading && !isError && schemas.length === 0;

  const helperText = noService
    ? 'Select a service first'
    : isError
      ? (error?.message ?? 'Failed to load schemas')
      : empty
        ? 'No schemas in this service'
        : undefined;

  if (allowCustom) {
    // Only a truly absent parent disables the control; a custom (free-typed)
    // parent keeps it enabled so the user can type a custom schema too.
    const parentMissing = noService && !parentIsCustom;
    return (
      <FreeSoloSelect<SchemaOption>
        name={name}
        label={label}
        options={schemas}
        getOptionLabel={getOptionName}
        required={required}
        disabled={disabled || parentMissing || isError}
        loading={isLoading}
        helperText={
          parentMissing
            ? 'Select a service first'
            : isError
              ? helperText
              : undefined
        }
        error={isError}
        noOptionsText={
          parentMissing
            ? 'Select a service first'
            : parentIsCustom
              ? 'Custom service — type a schema name'
              : isLoading
                ? 'Loading schemas…'
                : 'No schemas in this service'
        }
      />
    );
  }

  return (
    <AutoCompleteInput<SchemaOption>
      name={name}
      label={label}
      control={control}
      isRequired={required}
      loading={isLoading}
      disabled={disabled || noService || isError}
      options={schemas}
      controllerProps={{
        rules: required ? { required: `${label} is required` } : undefined,
      }}
      autoCompleteProps={{
        getOptionLabel,
        isOptionEqualToValue,
        noOptionsText: noService
          ? 'Select a service first'
          : isLoading
            ? 'Loading schemas…'
            : 'No schemas in this service',
      }}
      textFieldProps={{ helperText, error: isError }}
    />
  );
}
