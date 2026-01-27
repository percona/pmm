
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
import { Control, UseControllerProps } from 'react-hook-form';
import { LabeledContentProps } from '../../../labeled-content';

export type AutoCompleteInputProps<T> = {
  name: string;
  options: T[];
  control?: Control;
  controllerProps?: UseControllerProps;
  label?: string;
  labelProps?: LabeledContentProps;
  autoCompleteProps?: Omit<
    AutocompleteProps<
      T,
      boolean | undefined,
      boolean | undefined,
      boolean | undefined
    >,
    'options' | 'renderInput'
  >;
  textFieldProps?: TextFieldProps;
  loading?: boolean;
  isRequired?: boolean;
  disabled?: boolean;
  tooltipText?: string;
  onChange?: () => void;
};
