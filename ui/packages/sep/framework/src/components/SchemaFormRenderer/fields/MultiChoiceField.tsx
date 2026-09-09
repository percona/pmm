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

import { Controller, useFormContext, useWatch } from 'react-hook-form';
import Box from '@mui/material/Box';
import Checkbox from '@mui/material/Checkbox';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import { SchemaSelectShell } from '../SchemaSelectShell';
import type { MultiChoiceField as MultiChoiceFieldType } from '../types';
import { buildValidationRules } from '../utils/validationMapper';
import { renderChoiceLabel } from './choiceLabel';

interface MultiChoiceFieldProps {
  field: MultiChoiceFieldType;
}

export function MultiChoiceField({ field }: MultiChoiceFieldProps) {
  const { control } = useFormContext();
  const selected =
    (useWatch({ control, name: field.name }) as string[] | undefined) ?? [];
  const labelId = `${field.name}-label`;

  return (
    <Controller
      name={field.name}
      control={control}
      rules={buildValidationRules(field)}
      render={({ field: rhfField, fieldState: { error } }) => (
        <SchemaSelectShell
          field={rhfField}
          labelId={labelId}
          label={field.label}
          required={field.required}
          error={error}
          description={field.description}
          multiple
          renderValue={(value) => {
            const values = (value as string[] | undefined) ?? [];
            if (values.length === 0) {
              return (
                <Box component="span" sx={{ color: 'text.disabled' }}>
                  Select…
                </Box>
              );
            }
            return field.choices
              .filter((c) => values.includes(c.value))
              .map((c) => c.label)
              .join(', ');
          }}
        >
          {field.choices.map((choice) => (
            <MenuItem
              key={choice.value}
              value={choice.value}
              // Only block disabled options that are not already selected, so a
              // value that was selected before becoming disabled can still be
              // de-selected (a fully disabled MenuItem swallows the toggle).
              disabled={choice.disabled && !selected.includes(choice.value)}
            >
              <Checkbox checked={selected.includes(choice.value)} />
              <ListItemText primary={renderChoiceLabel(choice)} />
            </MenuItem>
          ))}
        </SchemaSelectShell>
      )}
    />
  );
}
