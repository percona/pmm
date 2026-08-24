import type { SvgIconProps } from '@mui/material/SvgIcon';
import type { DYNAMIC_ICON_IMPORT_MAP } from './Icon.constants';

export type IconName = keyof typeof DYNAMIC_ICON_IMPORT_MAP;

export interface IconProps extends SvgIconProps {
  name: IconName;
}
