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
import { AutocompleteProps, TextFieldProps } from '@mui/material';
import { Control, FieldPath, FieldValues, UseControllerProps } from 'react-hook-form';
import { LabeledContentProps } from '../../../labeled-content';

export type AutoCompleteInputProps<TOption, TFieldValues extends FieldValues = FieldValues> = {
  /** Form field the value is stored under; also drives the test ids. */
  name: FieldPath<TFieldValues>;
  /** Choices offered in the dropdown; the user can type to narrow them. */
  options: TOption[];
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<TFieldValues>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<TFieldValues>, 'name' | 'control'>;
  /** Floating label shown inside the field. */
  label?: string;
  /** Not used: this input renders the MUI floating label. Kept for API symmetry with `CheckboxInput` and `RadioGroup`. */
  labelProps?: LabeledContentProps;
  /** MUI Autocomplete props forwarded to the dropdown, such as multiple, freeSolo, or getOptionLabel. */
  autoCompleteProps?: Omit<
    AutocompleteProps<TOption, boolean | undefined, boolean | undefined, boolean | undefined>,
    'options' | 'renderInput'
  >;
  /** MUI TextField props forwarded to the text box, such as placeholder or helperText. */
  textFieldProps?: TextFieldProps;
  /** Shows a spinner inside the field while the choices are being fetched. */
  loading?: boolean;
  /** Marks the field required with an asterisk on the label; validation itself goes in `controllerProps.rules`. */
  isRequired?: boolean;
  /** Prevents the user from opening the dropdown or typing. */
  disabled?: boolean;
  /** Help text shown in a tooltip above the field on hover. */
  tooltipText?: string;
  /** Called after the user picks or clears a choice, once the form value is already updated. */
  onChange?: () => void;
};
