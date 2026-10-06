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
import { FormControlLabelProps as MuiFormControlLabelProps, SwitchProps } from '@mui/material';
import { Control, FieldPath, FieldValues, UseControllerProps } from 'react-hook-form';

type FormControlLabelProps = MuiFormControlLabelProps;

export type SwitchInputProps<T extends FieldValues = FieldValues> = {
  /** react-hook-form control; only needed when the input sits outside a `FormProvider`. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** MUI FormControlLabel props for the label row, such as labelPlacement. */
  formControlLabelProps?: Omit<FormControlLabelProps, 'control' | 'label'>;
  /** Form field the on/off state is stored under; also drives the test ids. */
  name: FieldPath<T>;
  /** Text shown next to the switch. */
  label: string;
  /** Smaller explanatory text shown under the label. */
  labelCaption?: string;
  /** MUI Switch props forwarded to the toggle, such as size or color; its onChange fires too. */
  switchFieldProps?: SwitchProps;
};
