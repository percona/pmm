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

import { get, useFormContext, useFormState } from 'react-hook-form';
import { DateTimeInput } from '../../DateTimeInput';
import { FieldLabelWithHelp } from '../FieldLabelWithHelp';
import { fieldHelp } from '../fieldHelp';
import type { DateTimeField as DateTimeFieldType } from '../types';
import { buildValidationRules } from '../utils/validationMapper';

interface DateTimeFieldProps {
  field: DateTimeFieldType;
}

export function DateTimeField({ field }: DateTimeFieldProps) {
  const { control } = useFormContext();
  // `DateTimeInput` owns the Controller, so the error has to be read from form
  // state rather than taken from a `fieldState` this component never sees.
  const { errors } = useFormState({ control, name: field.name });
  // Dotted paths nest; see SchemaSelectShell.
  const error = get(errors, field.name);
  const help = fieldHelp(field);
  return (
    <DateTimeInput
      name={field.name}
      control={control}
      label={
        <FieldLabelWithHelp label={field.label} description={help.tooltip} />
      }
      isRequired={field.required}
      error={!!error}
      helperText={(error?.message as string | undefined) ?? help.inline}
      controllerProps={{ rules: buildValidationRules(field) }}
    />
  );
}
