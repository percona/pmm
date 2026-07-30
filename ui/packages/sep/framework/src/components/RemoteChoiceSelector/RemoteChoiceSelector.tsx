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

import { useEffect, useMemo, useRef } from 'react';
import {
  Controller,
  useFormContext,
  useWatch,
  type ControllerRenderProps,
  type FieldError,
  type FieldValues,
} from 'react-hook-form';
import Autocomplete, { createFilterOptions } from '@mui/material/Autocomplete';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import TextField from '@mui/material/TextField';
import type { ChoiceOption } from '@sep/api';
import { useRemoteChoices } from '../../hooks/useRemoteChoices';
import { renderChoiceLabel } from '../SchemaFormRenderer/fields/choiceLabel';
import {
  normalizeChange,
  toDisplayValue,
  type ChoiceFreeSoloDisplayValue,
} from './choiceFreeSoloValue';

const EMPTY_OPTIONS: ChoiceOption[] = [];
const PARENT_MISSING_TEXT = 'Select a value first';

export interface RemoteChoiceSelectorProps {
  /** react-hook-form field name. Stores the committed `string | null` (option value, free-typed value, or unset). */
  name: string;
  label: string;
  required?: boolean;
  disabled?: boolean;
  /** Fully-resolved fetch path (relative to the `/api` base) the options load from. */
  endpointUrl: string;
  /** Optional parent field name; when set, the field cascades and stays disabled until the parent has a value. */
  dependsOn?: string;
  /** Offer free-text (free-solo) entry alongside the fetched options. */
  allowCustom?: boolean;
}

/** Normalize a cascade parent's watched value to a scalar string (or `null` when unset). */
function toScalarParent(value: unknown): string | null {
  if (value === null || value === undefined || value === '') {
    return null;
  }
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number') {
    return String(value);
  }
  if (typeof value === 'object') {
    const record = value as Record<string, unknown>;
    const identifier = record.value ?? record.id ?? record.name;
    return identifier === null || identifier === undefined
      ? null
      : String(identifier);
  }
  return null;
}

/**
 * Inner Autocomplete, split out so it can hold hooks driven by the Controller's
 * `field`. Renders `Choice`-compatible options (with disabled tooltips) and
 * commits a string `value` — never an inventory id.
 */
function RemoteChoiceAutocomplete({
  field,
  fieldError,
  label,
  required,
  disabled,
  allowCustom,
  options,
  loading,
  isError,
  helperText,
  noOptionsText,
}: {
  field: ControllerRenderProps<FieldValues, string>;
  fieldError?: FieldError;
  label: string;
  required?: boolean;
  disabled?: boolean;
  allowCustom: boolean;
  options: readonly ChoiceOption[];
  loading: boolean;
  isError: boolean;
  helperText?: string;
  noOptionsText: string;
}) {
  const filter = useMemo(
    () => createFilterOptions<ChoiceOption | string>(),
    []
  );
  const { value: fieldValue, onChange } = field;
  const resolved = toDisplayValue(fieldValue, options);
  // In non-freeSolo mode MUI expects an option object or null; a bare string
  // (an unloaded/unmatched persisted value) trips its "none of the options
  // match" warning, so collapse it to null unless free-text entry is enabled.
  const value = allowCustom || typeof resolved !== 'string' ? resolved : null;
  const isCustomValue = typeof value === 'string';
  const commit = (next: ChoiceFreeSoloDisplayValue) =>
    onChange(normalizeChange(next, options));

  return (
    <Autocomplete<ChoiceOption | string, false, false, boolean>
      freeSolo={allowCustom}
      options={options as (ChoiceOption | string)[]}
      value={value}
      disabled={disabled}
      loading={loading}
      forcePopupIcon
      // Keep freshly-typed text after blur so a custom value survives; a closed
      // select clears an unmatched entry instead.
      clearOnBlur={!allowCustom}
      selectOnFocus
      handleHomeEndKeys
      getOptionLabel={(option) =>
        typeof option === 'string' ? option : (option.label ?? '')
      }
      getOptionDisabled={(option) =>
        typeof option !== 'string' && Boolean(option.disabled)
      }
      isOptionEqualToValue={(a, b) =>
        typeof a !== 'string' && typeof b !== 'string' && a.value === b.value
      }
      noOptionsText={noOptionsText}
      data-testid={`${field.name}-autocomplete`}
      filterOptions={
        allowCustom
          ? (opts, params) => {
              const filtered = filter(opts, params);
              const input = params.inputValue.trim();
              const exists = options.some(
                (o) => o.label === input || o.value === input
              );
              if (input !== '' && !exists) {
                filtered.push(input);
              }
              return filtered;
            }
          : undefined
      }
      renderOption={(props, option) => {
        const { key, ...rest } = props as {
          key?: string;
        } & React.HTMLAttributes<HTMLLIElement>;
        return typeof option === 'string' ? (
          <Box component="li" key={key} {...rest} sx={{ fontStyle: 'italic' }}>
            <em>{option}</em>
          </Box>
        ) : (
          <Box component="li" key={key} {...rest}>
            {renderChoiceLabel(option)}
          </Box>
        );
      }}
      onChange={(_event, next) => commit(next)}
      onInputChange={(_event, input, reason) => {
        // Live typing → commit as a custom value (only meaningful in free-solo
        // mode). 'reset' / 'clear' are selection-driven and handled by onChange.
        if (allowCustom && reason === 'input') {
          commit(input);
        }
      }}
      renderInput={(params) => (
        <TextField
          {...params}
          inputRef={field.ref}
          onBlur={field.onBlur}
          label={label}
          required={required}
          error={isError || !!fieldError}
          helperText={fieldError ? fieldError.message : helperText}
          size="small"
          inputProps={{
            ...params.inputProps,
            style: isCustomValue ? { fontStyle: 'italic' } : undefined,
          }}
          InputProps={{
            ...params.InputProps,
            endAdornment: (
              <>
                {loading ? (
                  <CircularProgress color="inherit" size={20} />
                ) : null}
                {params.InputProps.endAdornment}
              </>
            ),
          }}
        />
      )}
    />
  );
}

/**
 * Generic selector for a `RemoteChoices` field: fetches `Choice`-compatible
 * options from the wire-declared `endpointUrl`, renders `value` / `label` /
 * `disabled` / `disabled_reason` exactly like a static `ChoiceField`, and — when
 * `allowCustom` is set — accepts a free-typed value. When `dependsOn` names a
 * parent field the fetch is parameterised by that field's value (as a query
 * parameter named after `dependsOn`) and the control stays disabled until the
 * parent has a value; the child value resets whenever the parent value changes.
 *
 * The committed react-hook-form value is always a string (an option `value` or a
 * free-typed value) or `null` — never an inventory id — so it flows to the task
 * payload identically to a static choice.
 */
export function RemoteChoiceSelector({
  name,
  label,
  required,
  disabled,
  endpointUrl,
  dependsOn,
  allowCustom,
}: RemoteChoiceSelectorProps) {
  const { control, setValue } = useFormContext();

  const parentRaw = useWatch({
    control,
    name: dependsOn ?? '',
    disabled: !dependsOn,
  });
  const parentScalar = dependsOn ? toScalarParent(parentRaw) : null;
  const cascades = dependsOn !== undefined;
  const parentMissing = cascades && parentScalar === null;

  const {
    data: options = EMPTY_OPTIONS,
    isLoading,
    isError,
    error,
  } = useRemoteChoices({
    endpointUrl,
    dependsOnName: dependsOn ?? null,
    dependsOnValue: parentScalar,
  });

  const parentKey = String(parentScalar);
  const prevParentKey = useRef(parentKey);
  useEffect(() => {
    if (cascades && prevParentKey.current !== parentKey) {
      prevParentKey.current = parentKey;
      setValue(name, null, { shouldDirty: true, shouldValidate: false });
    }
  }, [cascades, parentKey, name, setValue]);

  const empty =
    !parentMissing && !isLoading && !isError && options.length === 0;
  const helperText = parentMissing
    ? PARENT_MISSING_TEXT
    : isError
      ? (error?.message ?? 'Failed to load options')
      : empty
        ? 'No options available'
        : undefined;
  const noOptionsText = parentMissing
    ? PARENT_MISSING_TEXT
    : isLoading
      ? 'Loading…'
      : 'No options available';

  return (
    <Controller
      name={name}
      control={control}
      rules={required ? { required: `${label} is required` } : undefined}
      render={({ field, fieldState: { error: fieldError } }) => (
        <RemoteChoiceAutocomplete
          field={field}
          fieldError={fieldError}
          label={label}
          required={required}
          disabled={disabled || parentMissing || isError}
          allowCustom={Boolean(allowCustom)}
          options={options}
          loading={isLoading}
          isError={isError}
          helperText={helperText}
          noOptionsText={noOptionsText}
        />
      )}
    />
  );
}
