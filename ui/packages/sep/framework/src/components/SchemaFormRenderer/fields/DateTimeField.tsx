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

import { useFormContext } from 'react-hook-form';
import { TextInput } from '@percona/percona-ui';
import type { DateTimeField as DateTimeFieldType } from '../types';
import { buildValidationRules } from '../utils/validationMapper';

interface DateTimeFieldProps {
  field: DateTimeFieldType;
}

export function DateTimeField({ field }: DateTimeFieldProps) {
  const { control } = useFormContext();
  return (
    <TextInput
      name={field.name}
      label={field.label}
      isRequired={field.required}
      control={control}
      textFieldProps={{
        type: 'datetime-local',
        helperText: field.description,
        fullWidth: true,
        InputLabelProps: { shrink: true },
      }}
      controllerProps={{ rules: buildValidationRules(field) }}
    />
  );
}
