import { SvgIconProps } from '@mui/material';
import { ICON_MAP } from './Icon.constants';

export type IconName = keyof typeof ICON_MAP;

export interface IconProps extends SvgIconProps {
  name: IconName;
}
