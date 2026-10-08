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
import { SvgIconProps } from '@mui/material';

export interface StatusIconProps extends SvgIconProps {
  /** The size of the icon's canvas/boundary: large = 20px, small = 16px. Defaults to large. */
  size?: 'large' | 'small';
}

export interface IconsProps {
  /** CSS width the glyph is drawn at. */
  iconWidth: string;
  /** MUI SvgIcon props forwarded to the glyph. */
  props: SvgIconProps;
}

export interface StatusIconProviderProps {
  /** Glyph used while the app is in light mode. */
  LightIconGeneral: React.FC<IconsProps>;
  /** Glyph used while the app is in dark mode. */
  DarkIconGeneral: React.FC<IconsProps>;
  /** Size and MUI SvgIcon props passed on to the glyph. */
  props: StatusIconProps;
}
