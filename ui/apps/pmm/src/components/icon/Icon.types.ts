import { SvgIconProps } from '@mui/material/SvgIcon';
import { DYNAMIC_ICON_IMPORT_MAP } from './Icon.constants';

export type IconName = keyof typeof DYNAMIC_ICON_IMPORT_MAP;

export interface IconProps extends SvgIconProps {
  name: IconName;
}
