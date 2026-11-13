import { SvgIconProps } from '@mui/material';

export interface StatusIconProps extends SvgIconProps {
  /** Width of the icon: large = 20px, small = 16px. Defaults to large. */
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
