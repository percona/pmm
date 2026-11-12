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
import { ToggleButtonGroupProps } from '@mui/material';
import { LabeledContentProps } from '../../../labeled-content';
import { Control, FieldPath, FieldValues, UseControllerProps } from 'react-hook-form';

export type ToggleButtonGroupInputProps<T extends FieldValues = FieldValues> = {
  /** Form field the selected value is stored under; also drives the test id. */
  name: FieldPath<T>;
  /** Heading shown above the group; without it only the buttons render. */
  label?: string;
  /** Settings for the heading rendered by `LabeledContent`, such as a caption or required asterisk. */
  labelProps?: LabeledContentProps;
  /** react-hook-form control to bind to; the input still needs a `FormProvider` above it, because the selection is saved through the form context. */
  control?: Control<T>;
  /** Validation rules and other react-hook-form Controller settings for this field. */
  controllerProps?: Omit<UseControllerProps<T>, 'name' | 'control'>;
  /** MUI ToggleButtonGroup props, such as size or orientation; its onChange fires with the picked value. */
  toggleButtonGroupProps?: ToggleButtonGroupProps;
  /** The options, as MUI ToggleButton or ToggleCard elements; each one's `value` is what gets stored. */
  children: React.ReactNode;
};
