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
import { AutoCompleteInput } from '@percona/peak-ui';
import { useTables, type TableOption } from '../../hooks/useTables';
import type { SchemaOption } from '../../hooks/useSchemas';
import { extractId } from '../../utils/extractId';
import { FreeSoloSelect } from '../FreeSoloSelect';

const EMPTY_OPTIONS: TableOption[] = [];

export interface TableSelectorProps {
  /**
   * react-hook-form field name. Without `allowCustom`, stores a
   * `TableOption | null`; with `allowCustom`, stores the committed
   * `number | string | null` (inventory id, free-typed value, or unset).
   */
  name: string;
  label: string;
  required?: boolean;
  /**
   * Form field name of the parent `SchemaSelector`. The watched value is
   * either a `SchemaOption` or a raw schema id.
   */
  dependsOn: string;
  disabled?: boolean;
  /** Offer free-text (free-solo) entry alongside the inventory options. */
  allowCustom?: boolean;
}

const getOptionLabel = (opt: TableOption | string) =>
  typeof opt === 'string' ? opt : opt.name;
const getOptionName = (opt: TableOption) => opt.name;

const isOptionEqualToValue = (a: TableOption, b: TableOption) => a.id === b.id;

export function TableSelector({
  name,
  label,
  required,
  dependsOn,
  disabled,
  allowCustom,
}: TableSelectorProps) {
  const { control, setValue } = useFormContext();

  const parent = useWatch({ control, name: dependsOn }) as
    | SchemaOption
    | number
    | string
    | null
    | undefined;
  const parentText = typeof parent === 'string' ? parent.trim() : '';
  const parentIsCustom = Boolean(allowCustom) && parentText !== '';
  const schemaId = parentIsCustom ? null : extractId(parent);
  // A parent that resolves to no inventory id but holds a non-empty string is a
  // free-typed (custom) parent value, not an absent one. A free-solo child can
  // still accept a typed value in that case (it just has no options to offer).
  const parentResetKey = parentIsCustom
    ? `custom:${parentText}`
    : `id:${schemaId ?? 'none'}`;

  const prevParentKeyRef = useRef(parentResetKey);
  useEffect(() => {
    if (prevParentKeyRef.current !== parentResetKey) {
      prevParentKeyRef.current = parentResetKey;
      setValue(name, null, { shouldDirty: true, shouldValidate: false });
    }
  }, [parentResetKey, name, setValue]);

  const {
    data: tables = EMPTY_OPTIONS,
    isLoading,
    isError,
    error,
  } = useTables({ schemaId });

  const noSchema = schemaId === null || schemaId === undefined;
  const empty = !noSchema && !isLoading && !isError && tables.length === 0;

  const helperText = noSchema
    ? 'Select a schema first'
    : isError
      ? (error?.message ?? 'Failed to load tables')
      : empty
        ? 'No tables in this schema'
        : undefined;

  if (allowCustom) {
    // Only a truly absent parent disables the control; a custom (free-typed)
    // parent keeps it enabled so the user can type a custom table too.
    const parentMissing = noSchema && !parentIsCustom;
    return (
      <FreeSoloSelect<TableOption>
        name={name}
        label={label}
        options={tables}
        getOptionLabel={getOptionName}
        required={required}
        disabled={disabled || parentMissing || isError}
        loading={isLoading}
        helperText={
          parentMissing
            ? 'Select a schema first'
            : isError
              ? helperText
              : undefined
        }
        error={isError}
        noOptionsText={
          parentMissing
            ? 'Select a schema first'
            : parentIsCustom
              ? 'Custom schema — type a table name'
              : isLoading
                ? 'Loading tables…'
                : 'No tables in this schema'
        }
      />
    );
  }

  return (
    <AutoCompleteInput<TableOption>
      name={name}
      label={label}
      control={control}
      isRequired={required}
      loading={isLoading}
      disabled={disabled || noSchema || isError}
      options={tables}
      controllerProps={{
        rules: required ? { required: `${label} is required` } : undefined,
      }}
      autoCompleteProps={{
        getOptionLabel,
        isOptionEqualToValue,
        noOptionsText: noSchema
          ? 'Select a schema first'
          : isLoading
            ? 'Loading tables…'
            : 'No tables in this schema',
      }}
      textFieldProps={{ helperText, error: isError }}
    />
  );
}
