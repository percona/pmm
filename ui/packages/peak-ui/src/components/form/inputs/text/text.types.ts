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

import { FormHelperTextProps, TextFieldProps } from '@mui/material';
import { Control, FieldPath, FieldValues, UseControllerProps } from 'react-hook-form';

export type TextInputProps<T extends FieldValues = FieldValues> = {
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** Form field the text is stored under; also drives the test id. */
  name: FieldPath<T>;
  /** Label shown on the field's top border; it stays there even while the field is empty. */
  label?: string;
  /** MUI TextField props such as placeholder or multiline; an onBlur here must return the text to store, as its result replaces the value. */
  textFieldProps?: TextFieldProps;
  /** Marks the field required with an asterisk on the label; validation itself goes in `controllerProps.rules`. */
  isRequired?: boolean;
  /** Styles for the helper or error text under the field; only `sx` is applied. */
  formHelperTextProps?: Partial<FormHelperTextProps>;
};
