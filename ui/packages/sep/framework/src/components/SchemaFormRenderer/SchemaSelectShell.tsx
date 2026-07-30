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

import type { ReactNode } from 'react';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import Select, { type SelectProps } from '@mui/material/Select';
import type {
  ControllerRenderProps,
  FieldError,
  FieldValues,
} from 'react-hook-form';
import { FieldLabelWithHelp } from './FieldLabelWithHelp';

export interface SchemaSelectShellProps {
  /**
   * react-hook-form Controller field props, spread onto the `<Select>`. This
   * drives value/onChange and — because `name` flows through to the native
   * input — yields the MUI-derived `mui-component-select-${name}` test id.
   */
  field: ControllerRenderProps<FieldValues, string>;
  /** Id linking `<InputLabel>` to `<Select>` for `aria-labelledby`. */
  labelId: string;
  label: string;
  required?: boolean;
  /** rhf field error; presence flips `aria-invalid` and the outline color. */
  error?: FieldError;
  /** Helper text shown when there is no error message. */
  description?: ReactNode;
  /** Render a multi-select (value is an array). */
  multiple?: boolean;
  /** Owns the empty-placeholder vs populated branch — differs per field. */
  renderValue: SelectProps<unknown>['renderValue'];
  /** MenuItem children. */
  children: ReactNode;
}

/**
 * Shared scaffold for the schema-driven select fields (MultiChoiceField and
 * ChoiceField's select-mode branch). Owns the
 * FormControl → InputLabel → Select → FormHelperText composition with the
 * fixed-state knobs (`shrink`, `notched`, `displayEmpty`, `fullWidth`,
 * `size="small"`) established in SEP-1278. The per-field `renderValue` and
 * MenuItem content stay at the call site.
 */
export function SchemaSelectShell({
  field,
  labelId,
  label,
  required,
  error,
  description,
  multiple,
  renderValue,
  children,
}: SchemaSelectShellProps) {
  const helpDescription =
    typeof description === 'string' ? description : undefined;

  return (
    <FormControl fullWidth size="small" error={!!error}>
      <InputLabel id={labelId} shrink required={required}>
        <FieldLabelWithHelp label={label} description={helpDescription} />
      </InputLabel>
      <Select
        {...field}
        labelId={labelId}
        label={label}
        multiple={multiple}
        displayEmpty
        notched
        data-testid={`select-${field.name}-button`}
        inputProps={{ 'data-testid': `select-input-${field.name}` }}
        renderValue={renderValue}
      >
        {children}
      </Select>
      {(error?.message || description) && (
        <FormHelperText>{error?.message ?? description}</FormHelperText>
      )}
    </FormControl>
  );
}
