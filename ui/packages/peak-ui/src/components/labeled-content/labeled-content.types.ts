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

import { SxProps, Theme, TypographyProps } from '@mui/material';

export type LabeledContentProps = {
  /** Section heading shown above the content. */
  label?: string;
  /** Explanatory text shown under the heading. */
  caption?: string;
  /** Not used: pass the content as `children` instead. */
  verticalStackChildrenSlot?: React.ReactNode;
  /** Content shown on the heading row, to the right of the label, such as a button or chip. */
  horizontalStackChildrenSlot?: React.ReactNode;
  /** Adds a required asterisk after the heading. */
  isRequired?: boolean;
  /** Styles for the whole block: heading, caption, and children stacked vertically. */
  verticalStackSx?: SxProps<Theme>;
  /** Styles for the heading row. */
  horizontalStackSx?: SxProps<Theme>;
} & TypographyProps;
