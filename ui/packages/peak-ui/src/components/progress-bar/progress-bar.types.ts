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
export type ProgressBarProps = {
  /** Prefix for the test ids of the bar and its label. */
  dataTestId?: string;
  /** Amount already used, drawn as the solid part of the bar; hidden when `buffer` exceeds `total`. */
  value: number;
  /** Amount reserved or pending, drawn as a contrasting segment after the used part; above `total`, the whole bar turns warning-colored. */
  buffer: number;
  /** Amount that fills the bar completely. */
  total: number;
  /** Text shown above the bar, aligned right, such as "12 of 20 GB". */
  label: string;
};
