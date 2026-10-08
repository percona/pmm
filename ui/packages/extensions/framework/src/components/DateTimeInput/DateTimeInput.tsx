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

import { useRef, type ReactNode } from 'react';
import { LocalizationProvider } from '@mui/x-date-pickers';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFnsV3';
import { DateTimePickerInput } from '@percona/peak-ui';
import type { Control, FieldValues, UseControllerProps } from 'react-hook-form';
import { dateToWallClock, wallClockToDate } from './wallClockValue';

export interface DateTimeInputProps<T extends FieldValues = FieldValues> {
  name: string;
  control: Control<T>;
  label: ReactNode;
  isRequired?: boolean;
  helperText?: ReactNode;
  error?: boolean;
  disabled?: boolean;
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
}

/**
 * The framework's one date-time control: Peak UI's picker over a wall-clock
 * string.
 *
 * Two things it owns that a bare `DateTimePickerInput` would not.
 *
 * It carries its own `LocalizationProvider`. The picker needs a date adapter in
 * context, and `@pmm-extensions/framework` is consumed as source by whatever app mounts
 * it — including tests that render a single field. Depending on every host to
 * supply one would make the control throw in exactly the places it is easiest
 * to forget. Nesting is harmless where the host already has one (PMM's
 * `App.tsx` does).
 *
 * It fixes the value shape at a wall-clock string rather than a `Date`. See
 * `wallClockValue.ts` for why the callers need that.
 */
export function DateTimeInput<T extends FieldValues = FieldValues>({
  name,
  control,
  label,
  isRequired,
  helperText,
  error,
  disabled,
  controllerProps,
}: DateTimeInputProps<T>) {
  // The output transform maps a half-typed date to '', which an optional field
  // would submit as "no value"; the picker's own error keeps it from submitting.
  const pickerInvalid = useRef(false);
  const rules = controllerProps?.rules;
  const validate =
    typeof rules?.validate === 'function'
      ? { schema: rules.validate }
      : rules?.validate;

  return (
    <LocalizationProvider dateAdapter={AdapterDateFns}>
      <DateTimePickerInput
        name={name as never}
        control={control}
        controllerProps={{
          ...controllerProps,
          rules: {
            ...rules,
            validate: {
              ...validate,
              pickerValid: () =>
                !pickerInvalid.current || 'Enter a valid date and time',
            },
          },
        }}
        onError={(pickerError) => {
          pickerInvalid.current = pickerError !== null;
        }}
        label={label}
        disabled={disabled}
        transform={{
          input: wallClockToDate,
          output: dateToWallClock,
        }}
        slotProps={{
          textField: {
            fullWidth: true,
            size: 'small',
            required: isRequired,
            error,
            helperText,
          },
        }}
      />
    </LocalizationProvider>
  );
}
