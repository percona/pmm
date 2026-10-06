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
import { RadioGroupProps as MuiRadioGroupProps, RadioProps } from '@mui/material';
import { Control, FieldPath, FieldValues, UseControllerProps } from 'react-hook-form';
import { LabeledContentProps } from '../../../labeled-content';

export type RadioGroupOptions = {
  /** Text shown next to the radio. */
  label: string;
  /** Value stored in the form when this option is picked. */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value: any;
  /** Greys out this option so it cannot be picked. */
  disabled?: boolean;
  /** MUI Radio props forwarded to this option's radio, such as size or color. */
  radioProps?: RadioProps;
};

export type RadioGroupProps<T extends FieldValues = FieldValues> = {
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** Form field the picked value is stored under; also drives the test id. */
  name: FieldPath<T>;
  /** Heading shown above the options; without it only the radios render. */
  label?: string;
  /** Settings for the heading rendered by `LabeledContent`, such as a caption. */
  labelProps?: LabeledContentProps;
  /** MUI RadioGroup props; pass `{ row: false }` to stack the options vertically. */
  radioGroupFieldProps?: MuiRadioGroupProps;
  /** Adds a required asterisk to the heading; validation itself goes in `controllerProps.rules`. */
  isRequired?: boolean;
  /** Choices to show, one radio each. */
  options: RadioGroupOptions[];
};
