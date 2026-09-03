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
import { TextInput } from '@percona/peak-ui';
import { FieldLabelWithHelp } from '../FieldLabelWithHelp';
import type { YamlField as YamlFieldType } from '../types';
import { buildValidationRules } from '../utils/validationMapper';

interface YamlFieldProps {
  field: YamlFieldType;
}

export function YamlField({ field }: YamlFieldProps) {
  const { control } = useFormContext();
  return (
    <TextInput
      name={field.name}
      label={field.label}
      isRequired={field.required}
      control={control}
      textFieldProps={{
        label: (
          <FieldLabelWithHelp
            label={field.label}
            description={field.description}
          />
        ),
        multiline: true,
        rows: field.rows ?? 8,
        placeholder: field.placeholder,
        helperText: field.description,
        fullWidth: true,
        inputProps: {
          style: { fontFamily: "'Roboto Mono', monospace", fontSize: 13 },
          spellCheck: false,
        },
      }}
      controllerProps={{ rules: buildValidationRules(field) }}
    />
  );
}
