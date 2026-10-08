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

import { LabeledContentProps } from '../../../labeled-content';
import { FormControlProps, SelectProps } from '@mui/material';
import { Control, FieldPath, FieldValues, UseControllerProps } from 'react-hook-form';

export type SelectInputProps<T extends FieldValues = FieldValues> = {
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** Form field the selected value is stored under; also drives the test ids. */
  name: FieldPath<T>;
  /** Floating label shown inside the field or at the top, depending on state. */
  label?: string;
  /** Hint shown under the field. Validation errors only turn the field red, so pass the message here. */
  helperText?: React.ReactNode;
  /** Not used: this input renders the MUI floating label. Kept for API symmetry with the inputs that use `LabeledContent`. */
  labelProps?: LabeledContentProps;
  /** MUI Select props forwarded to the dropdown, such as multiple or renderValue. */
  selectFieldProps?: SelectProps;
  /** MUI FormControl props for the wrapper, such as fullWidth or size (defaults to small). */
  formControlProps?: FormControlProps;
  /** The choices, as MUI MenuItem elements; with none, a disabled "No options" item is shown. */
  children?: React.ReactNode | React.ReactNode[];
  /** Accepted but not rendered today: no required marker is shown. Put required validation in `controllerProps.rules`. */
  isRequired?: boolean;
  /** Replaces the dropdown arrow with a spinner while the choices are being fetched. */
  loading?: boolean;
};
