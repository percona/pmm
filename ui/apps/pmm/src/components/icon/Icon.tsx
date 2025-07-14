import { FC, memo, lazy, Suspense } from 'react';
import { IconProps } from './Icon.types';
import { DYNAMIC_ICON_IMPORT_MAP, VIEWBOX_MAP } from './Icon.constants';
import SvgIcon from '@mui/material/SvgIcon';

const Icon: FC<IconProps> = memo(({ name, ...props }) => {
  if (!DYNAMIC_ICON_IMPORT_MAP[name]) {
    return null;
  }

  const Icon = lazy(DYNAMIC_ICON_IMPORT_MAP[name]);

  return (
    <Suspense fallback={null}>
      <SvgIcon component={Icon} viewBox={VIEWBOX_MAP[name]} {...props} />
    </Suspense>
  );
});

export default Icon;
