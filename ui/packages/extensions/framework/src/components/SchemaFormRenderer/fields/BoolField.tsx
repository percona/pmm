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

import { useId } from 'react';
import { get, useFormContext, useWatch } from 'react-hook-form';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import FormHelperText from '@mui/material/FormHelperText';
import { SwitchInput } from '@percona/peak-ui';
import { FieldHelpIcon } from '../FieldLabelWithHelp';
import { fieldHelp } from '../fieldHelp';
import type { BoolField as BoolFieldType } from '../types';

interface BoolFieldProps {
  field: BoolFieldType;
}

export function BoolField({ field }: BoolFieldProps) {
  const { control, formState } = useFormContext();
  const error = get(formState.errors, field.name);
  const errorId = useId();
  const help = fieldHelp(field);
  const value = useWatch({ control, name: field.name });

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 0.5 }}>
        <Box sx={{ minWidth: 0 }}>
          <SwitchInput
            name={field.name}
            label={field.label}
            labelCaption={help.inline}
            control={control}
            switchFieldProps={{
              slotProps: {
                input: {
                  'aria-invalid': Boolean(error),
                  'aria-describedby': error ? errorId : undefined,
                },
              },
            }}
          />
          {error && (
            <FormHelperText error id={errorId}>
              {error.message}
            </FormHelperText>
          )}
        </Box>
        {help.tooltip ? (
          <FieldHelpIcon description={help.tooltip} label={field.label} />
        ) : null}
      </Box>
      {field.destructive && value && (
        <Alert
          severity="warning"
          sx={{ mt: 1 }}
          data-testid="destructive-warning"
        >
          {field.destructive}
        </Alert>
      )}
    </Box>
  );
}
