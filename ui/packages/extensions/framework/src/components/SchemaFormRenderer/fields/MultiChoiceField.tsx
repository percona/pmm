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

import { useFormContext, useWatch } from 'react-hook-form';
import Checkbox from '@mui/material/Checkbox';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import { SchemaSelectShell } from '../SchemaSelectShell';
import type { MultiChoiceField as MultiChoiceFieldType } from '../types';
import { fieldHelp } from '../fieldHelp';
import { buildValidationRules } from '../utils/validationMapper';
import { renderChoiceLabel } from './choiceLabel';

interface MultiChoiceFieldProps {
  field: MultiChoiceFieldType;
}

export function MultiChoiceField({ field }: MultiChoiceFieldProps) {
  const { control } = useFormContext();
  const help = fieldHelp(field);
  const selected =
    (useWatch({ control, name: field.name }) as string[] | undefined) ?? [];

  return (
    <SchemaSelectShell
      name={field.name}
      label={field.label}
      required={field.required}
      rules={buildValidationRules(field)}
      tooltip={help.tooltip}
      inline={help.inline}
      multiple
      renderValue={(value) => {
        const values = (value as string[] | undefined) ?? [];
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
  );
}
