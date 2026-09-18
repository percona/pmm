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
import Box from '@mui/material/Box';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormHelperText from '@mui/material/FormHelperText';
import MenuItem from '@mui/material/MenuItem';
import MuiRadioGroup from '@mui/material/RadioGroup';
import Radio from '@mui/material/Radio';
import { LabeledContent, RadioGroup } from '@percona/percona-ui';
import { SchemaSelectShell } from '../SchemaSelectShell';
import type { ChoiceField as ChoiceFieldType } from '../types';
import { buildValidationRules } from '../utils/validationMapper';
import { renderChoiceLabel } from './choiceLabel';

interface ChoiceFieldProps {
  field: ChoiceFieldType;
}

// Small option sets render better as radios; larger as a select.
const RADIO_THRESHOLD = 3;

export function ChoiceField({ field }: ChoiceFieldProps) {
  const { control } = useFormContext();
  const rules = buildValidationRules(field);
  const hasDisabledChoice = field.choices.some((choice) => choice.disabled);

  if (field.choices.length > 0 && field.choices.length <= RADIO_THRESHOLD) {
    // The percona-ui RadioGroup keys its options by `label`, so a
    // Tooltip-wrapped (ReactNode) label would collide to the same React key
    // across multiple disabled options. Only the disabled-aware path needs
    // ReactNode labels, so render that branch with MUI primitives keyed by
    // `value`; the common path stays on the shared percona-ui RadioGroup
    // unchanged.
    if (hasDisabledChoice) {
      return (
        <LabeledContent
          label={field.label}
          isRequired={field.required}
          caption={field.description}
        >
          <Controller
            name={field.name}
            control={control}
            rules={rules}
            render={({ field: rhfField, fieldState: { error } }) => (
              <FormControl error={!!error} component="fieldset">
                <MuiRadioGroup row aria-label={field.label} {...rhfField}>
                  {field.choices.map((choice) => (
                    <FormControlLabel
                      key={choice.value}
                      value={choice.value}
                      disabled={choice.disabled}
                      control={<Radio />}
                      label={renderChoiceLabel(choice)}
                    />
                  ))}
                </MuiRadioGroup>
                {error?.message && (
                  <FormHelperText>{error.message}</FormHelperText>
                )}
              </FormControl>
            )}
          />
        </LabeledContent>
      );
    }

    return (
      <RadioGroup
        name={field.name}
        label={field.label}
        isRequired={field.required}
        control={control}
        labelProps={{ caption: field.description }}
        options={field.choices}
        controllerProps={{ rules }}
      />
    );
  }

  const labelId = `${field.name}-label`;

  return (
    <Controller
      name={field.name}
      control={control}
      rules={rules}
      render={({ field: rhfField, fieldState: { error } }) => (
        <SchemaSelectShell
          field={rhfField}
          labelId={labelId}
          label={field.label}
          required={field.required}
          error={error}
          description={field.description}
          renderValue={(value) => {
            if (value === undefined || value === null || value === '') {
              return (
                <Box component="span" sx={{ color: 'text.disabled' }}>
                  Select…
                </Box>
              );
            }
            return (
              field.choices.find((c) => c.value === value)?.label ??
              String(value)
            );
          }}
        >
          {field.choices.map((choice) => (
            <MenuItem
              key={choice.value}
              value={choice.value}
              disabled={choice.disabled}
            >
              {renderChoiceLabel(choice)}
            </MenuItem>
          ))}
        </SchemaSelectShell>
      )}
    />
  );
}
