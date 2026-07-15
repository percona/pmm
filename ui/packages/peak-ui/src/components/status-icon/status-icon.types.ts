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
import { type BoxProps } from '@mui/material/Box';

export type StatusIconSeverity = 'success' | 'info' | 'indeterminate' | 'warning' | 'error';

export type StatusIconSize = 'medium' | 'small';

export interface StatusIconProps extends Omit<BoxProps, 'color' | 'children'> {
  /** Semantic status. Determines both the color and the (locked) glyph. */
  severity: StatusIconSeverity;
  /** medium = 32px (symbol inside a tinted circular wrapper), small = 22px (bare symbol). */
  size?: StatusIconSize;
  /** Elevation surface the icon sits on; backs the small mask and medium base so translucent surfaces composite. Defaults to theme paper. */
  surfaceColor?: string;
  /** Accessible name. When set, the icon is exposed as an image; otherwise it stays decorative (aria-hidden). */
  titleAccess?: string;
}
