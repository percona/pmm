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

import { useEffect, useMemo } from 'react';
import {
  useFormContext,
  Controller,
  type ControllerRenderProps,
  type FieldError,
  type FieldValues,
} from 'react-hook-form';
import Autocomplete, { createFilterOptions } from '@mui/material/Autocomplete';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import TextField from '@mui/material/TextField';
import {
  toDisplayValue,
  normalizeChange,
  type ReferenceOption,
  type FreeSoloDisplayValue,
} from './freeSoloValue';

export interface FreeSoloSelectProps<T extends ReferenceOption> {
  /** react-hook-form field name. Stores `number | string | null` (an inventory id, a free-typed value, or unset). */
  name: string;
  label: string;
  options: readonly T[];
  /** Label shown for an inventory option (e.g. `name`, or `name (type)`). */
  getOptionLabel: (option: T) => string;
  required?: boolean;
  disabled?: boolean;
  loading?: boolean;
  helperText?: string;
  error?: boolean;
  noOptionsText?: string;
  onOpen?: () => void;
}

const isOptionEqualToValue = <T extends ReferenceOption>(
  option: T | string,
  value: T | string
): boolean =>
  typeof option !== 'string' &&
  typeof value !== 'string' &&
  option.id === value.id;

/**
 * Inner Autocomplete, split out so it can hold hooks (`useEffect`/`useMemo`)
 * driven by the Controller's `field`.
 */
function FreeSoloAutocomplete<T extends ReferenceOption>({
  field,
  fieldError,
  label,
  options,
  getOptionLabel,
  required,
  disabled,
  loading,
  helperText,
  error,
  noOptionsText,
  onOpen,
}: Omit<FreeSoloSelectProps<T>, 'name'> & {
  field: ControllerRenderProps<FieldValues, string>;
  fieldError?: FieldError;
}) {
  const filter = useMemo(() => createFilterOptions<T | string>(), []);

  const labelOf = (option: T | string): string =>
    typeof option === 'string' ? option : getOptionLabel(option);

  // Resolve a stored free string to the matching option id once the options
  // arrive, so it renders as the option label rather than raw text — regardless
  // of load timing. Two cases resolve:
  //   - an exact label match (a value typed before options loaded);
  //   - a string that matches an option id (stringified numeric inventory ids
  //     like `"4"`, or string host ids like `"nomad-1"`).
  const { value: fieldValue, onChange } = field;
  useEffect(() => {
    if (typeof fieldValue === 'string' && fieldValue.trim() !== '') {
      const trimmed = fieldValue.trim();
      const labelMatch = options.find((o) => getOptionLabel(o) === trimmed);
      if (labelMatch) {
        onChange(labelMatch.id);
        return;
      }
      const idMatch = options.find(
        (o) => o.id === trimmed || String(o.id) === trimmed
      );
      if (idMatch) {
        onChange(idMatch.id);
      }
    }
    // Intentionally keyed on `options` only: re-resolve a stored string when the
    // option set changes, without re-running on every keystroke.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [options]);

  const value = toDisplayValue<T>(fieldValue, options);
  const isCustomValue = typeof value === 'string';
  const commit = (next: FreeSoloDisplayValue<T>) =>
    onChange(normalizeChange<T>(next, options, getOptionLabel));

  return (
    <Autocomplete<T | string, false, false, true>
      freeSolo
      options={options as (T | string)[]}
      value={value}
      disabled={disabled}
      loading={loading}
      forcePopupIcon
      // Keep freshly-typed text after blur so a custom value survives.
      clearOnBlur={false}
      selectOnFocus
      handleHomeEndKeys
      getOptionLabel={labelOf}
      isOptionEqualToValue={isOptionEqualToValue}
      noOptionsText={noOptionsText}
      onOpen={onOpen}
      data-testid={`${field.name}-autocomplete`}
      filterOptions={(opts, params) => {
        const filtered = filter(opts, params);
        const input = params.inputValue.trim();
        // Offer the typed text as a "create" suggestion unless it already
        // matches an inventory option's label.
        const exists = options.some((o) => getOptionLabel(o) === input);
        if (input !== '' && !exists) {
          filtered.push(input);
        }
        return filtered;
      }}
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
            {getOptionLabel(option)}
          </Box>
        );
      }}
      onChange={(_event, next) => commit(next)}
      onInputChange={(_event, input, reason) => {
        // Live typing → commit as a custom value (or resolve to an id when it
        // matches an option). 'reset' / 'clear' are selection-driven and handled
        // by onChange, so ignore them here.
        if (reason === 'input') {
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
          error={error || !!fieldError}
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
 * Free-solo (`allow_custom`) variant of a reference selector: a single
 * MUI Autocomplete combo that lets the user pick an inventory option or type a
 * new value. Picking an option (or typing an exact label match) commits the
 * inventory **id**; typing a novel value commits the **string**; clearing
 * commits `null`.
 *
 * The committed value round-trips through react-hook-form so the conditional
 * and validation engines keep firing, and `coerceFormValues` passes the scalar
 * through unchanged on submit. A typed (non-inventory) value is rendered in
 * italics — both as the create suggestion in the dropdown and in the input —
 * so it reads as distinct from inventory options.
 */
export function FreeSoloSelect<T extends ReferenceOption>({
  name,
  required,
  ...rest
}: FreeSoloSelectProps<T>) {
  const { control } = useFormContext();

  return (
    <Controller
      name={name}
      control={control}
      rules={required ? { required: `${rest.label} is required` } : undefined}
      render={({ field, fieldState: { error: fieldError } }) => (
        <FreeSoloAutocomplete<T>
          field={field}
          fieldError={fieldError}
          required={required}
          {...rest}
        />
      )}
    />
  );
}
