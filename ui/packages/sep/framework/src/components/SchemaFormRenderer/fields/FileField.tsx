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

import { Controller, useFormContext } from 'react-hook-form';
import { IconButton, InputAdornment, TextField } from '@mui/material';
import AttachFileIcon from '@mui/icons-material/AttachFile';
import { FieldLabelWithHelp } from '../FieldLabelWithHelp';
import type { FileField as FileFieldType } from '../types';
import { buildValidationRules } from '../utils/validationMapper';

interface FileFieldProps {
  field: FileFieldType;
}

// percona-ui's FileInput owns its own internal Controller and exposes no
// controllerProps, so we cannot attach react-hook-form rules to it. The form
// also sets noValidate, which suppresses the native required attribute. To
// make required (and any future file rules) actually block submission, we
// render the Controller ourselves here.
export function FileField({ field }: FileFieldProps) {
  const { control } = useFormContext();
  const rules = buildValidationRules(field);
  const inputId = `file-input-${field.name}`;

  return (
    <Controller
      name={field.name}
      control={control}
      rules={rules}
      render={({ field: rhf, fieldState: { error } }) => (
        <TextField
          label={
            <FieldLabelWithHelp
              label={field.label}
              description={field.description}
            />
          }
          required={field.required}
          fullWidth
          size="small"
          value={rhf.value instanceof File ? rhf.value.name : ''}
          error={!!error}
          helperText={error ? error.message : field.description}
          slotProps={{
            input: {
              readOnly: true,
              endAdornment: (
                <InputAdornment position="end">
                  <IconButton
                    aria-label={`Select file for ${field.label}`}
                    component="label"
                    htmlFor={inputId}
                    edge="end"
                  >
                    <AttachFileIcon fontSize="small" />
                    <input
                      id={inputId}
                      type="file"
                      hidden
                      accept={field.accept?.join(',')}
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        rhf.onChange(file ?? undefined);
                      }}
                      onBlur={rhf.onBlur}
                    />
                  </IconButton>
                </InputAdornment>
              ),
            },
          }}
        />
      )}
    />
  );
}
