import { FC, memo } from 'react';
import { IconProps } from './Icon.types';
import { ICON_MAP } from './Icon.constants';
import { SvgIcon } from '@mui/material';

const Icon: FC<IconProps> = memo(({ name, ...props }) => {
  const Icon = ICON_MAP[name];

  return <SvgIcon component={Icon} {...props} />;
});

export default Icon;
