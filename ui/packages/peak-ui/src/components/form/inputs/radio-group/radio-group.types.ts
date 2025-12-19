// everest
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
import {
  RadioGroupProps as MuiRadioGroupProps,
  RadioProps,
} from '@mui/material';
import { Control, UseControllerProps } from 'react-hook-form';
import { LabeledContentProps } from '../../../labeled-content';

export type RadioGroupOptions = {
  label: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value: any;
  disabled?: boolean;
  radioProps?: RadioProps;
};

export type RadioGroupProps = {
  control?: Control;
  controllerProps?: Omit<UseControllerProps, 'name'>;
  name: string;
  label?: string;
  labelProps?: LabeledContentProps;
  radioGroupFieldProps?: MuiRadioGroupProps;
  children?: React.ReactNode;
  isRequired?: boolean;
  options: RadioGroupOptions[];
};
