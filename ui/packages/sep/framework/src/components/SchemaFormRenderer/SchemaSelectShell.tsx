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
import { SelectInput } from '@percona/peak-ui';
import {
  get,
  useFormContext,
  useFormState,
  type RegisterOptions,
} from 'react-hook-form';
import type { SelectProps } from '@mui/material/Select';
import { FieldLabelWithHelp } from './FieldLabelWithHelp';

export interface SchemaSelectShellProps {
  /** Form field name; also the key `SelectInput` derives its test ids from. */
  name: string;
  label: string;
  required?: boolean;
  /** react-hook-form rules, applied by the Controller `SelectInput` owns. */
  rules?: RegisterOptions;
  /** Shown behind the help icon beside the label. */
  tooltip?: string;
  /** Shown under the control, unless an error takes the slot. */
  inline?: string;
  /** Render a multi-select (value is an array). */
  multiple?: boolean;
  /** Formats the closed control's contents — differs per field. */
  renderValue?: SelectProps<unknown>['renderValue'];
  /** MenuItem children. */
  children: ReactNode;
}

/**
 * Shared scaffold for the schema-driven select fields (MultiChoiceField and
 * ChoiceField's select-mode branch), over Peak UI's `SelectInput`.
 *
 * Deliberately plain: no `displayEmpty`, no forced `notched`/`shrink`, and no
 * "Select…" placeholder. Those were what made a select look unlike every other
 * control on the same form — a label pinned into the outline notch beside text
 * inputs whose labels sit *in* the empty field. Letting the label float gives
 * the app one label style, and an empty select reads as its own label, the way
 * `TextInput` and `AutoCompleteInput` already do (PMM-15456).
 *
 * `SelectInput` owns the Controller, so callers pass `name`/`rules` rather than
 * wrapping this in one of their own; it wires the error state from form state
 * but not the message, which is why the helper text is resolved here.
 */
export function SchemaSelectShell({
  name,
  label,
  required,
  rules,
  tooltip,
  inline,
  multiple,
  renderValue,
  children,
}: SchemaSelectShellProps) {
  const { control } = useFormContext();
  const { errors } = useFormState({ control, name });
  // `get`, not `errors[name]`: a one-of branch field's name is a dotted path
  // (`source.mode`) and react-hook-form nests its error to match, so a literal
  // lookup would leave the field outlined red with no message under it.
  const message = get(errors, name)?.message as string | undefined;

  return (
    <SelectInput
      name={name}
      control={control}
      controllerProps={{ rules }}
      label={
        (<FieldLabelWithHelp label={label} description={tooltip} />) as never
      }
      helperText={message ?? inline}
      formControlProps={{ fullWidth: true, size: 'small', required }}
      selectFieldProps={{ multiple, renderValue }}
    >
      {children}
    </SelectInput>
  );
}
