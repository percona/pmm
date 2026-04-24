// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { useEffect, useState } from 'react';
import { FieldError, useFieldArray, useFormContext } from 'react-hook-form';
import { IconButton, InputAdornment } from '@mui/material';
import { DeleteOutlineOutlined as DeleteOutlineOutlinedIcon } from '@mui/icons-material';

import { TextArrayProps } from './text-array.types';
import TextInput from '../text';
import ActionableLabeledContent from '../../../actionable-labeled-content';

const TextArray = ({
  fieldName,
  fieldKey,
  label,
  placeholder,
  control: controlProp,
  handleBlur,
  onRemove = () => {},
}: TextArrayProps) => {
  const formContext = useFormContext();
  const control = controlProp ?? formContext?.control;
  const errors = formContext?.formState?.errors;
  const watch = formContext?.watch;
  const trigger = formContext?.trigger;

  const { fields, append, remove } = useFieldArray({
    control,
    name: fieldName,
  });
  const defaultFields: Record<string, string>[] = fields.length ? fields : [];
  const [fieldAppended, setFieldAppended] = useState(false);
  const error = (index: number): FieldError | undefined =>
    // @ts-expect-error react-hook-form errors path is not strongly typed for nested field arrays
    errors?.[fieldName]?.[index]?.[fieldKey];

  useEffect(() => {
    if (fieldAppended) {
      const lastField = document.querySelector(
        `input[name="${fieldName}.${fields.length - 1}.${fieldKey}"]`
      );
      if (lastField) {
        (lastField as HTMLInputElement).focus();
      }
      setFieldAppended(false);
    }
  }, [fieldAppended]);

  return (
    <>
      <ActionableLabeledContent
        label={label}
        actionButtonProps={{
          dataTestId: 'add-text-input-button',
          onClick: () => {
            append({
              [fieldKey]: '',
            });
            setFieldAppended(true);
          },
        }}
      >
        {defaultFields.map((field, index) => (
          <TextInput
            control={control}
            name={`${fieldName}.${index}.${fieldKey}`}
            key={`${fieldName}_${field.id}`}
            textFieldProps={{
              variant: 'outlined',
              placeholder,
              error: !!error(index),
              helperText: error(index) ? error(index)?.message : '',
              sx: {
                width: '100%',
                '&:not(:last-child)': {
                  mb: 1,
                },
              },

              InputProps: {
                onChange: () => {
                  const initialFieldValue = watch?.(`${fieldName}.${index}.${fieldKey}`) || [];
                  defaultFields[index][fieldKey] = initialFieldValue;
                },
                onBlur: handleBlur
                  ? (e) => {
                      handleBlur(
                        e.target.value,
                        `${fieldName}.${index}.${fieldKey}`,
                        !!error(index)
                      );
                    }
                  : undefined,
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      data-testid={`delete-text-input-${index}-button`}
                      onClick={() => {
                        remove(index);
                        onRemove(index);
                        trigger?.(fieldName);
                      }}
                    >
                      <DeleteOutlineOutlinedIcon />
                    </IconButton>
                  </InputAdornment>
                ),
              },
            }}
          />
        ))}
      </ActionableLabeledContent>
    </>
  );
};

export default TextArray;
