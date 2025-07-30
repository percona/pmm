import { FC, memo, lazy, Suspense } from 'react';
import { IconProps } from './Icon.types';
import { DYNAMIC_ICON_IMPORT_MAP, VIEWBOX_MAP } from './Icon.constants';
import SvgIcon from '@mui/material/SvgIcon';
import Box from '@mui/material/Box';

const Icon: FC<IconProps> = memo(({ name, ...props }) => {
  if (!DYNAMIC_ICON_IMPORT_MAP[name]) {
    return null;
  }

  const Icon = lazy(DYNAMIC_ICON_IMPORT_MAP[name]);

  return (
    <Suspense
      fallback={
        <Box
          sx={{
            width: props.width || 24,
            height: props.height || 24,
            ...props.sx,
          }}
        />
      }
    >
      <SvgIcon component={Icon} viewBox={VIEWBOX_MAP[name]} {...props} />
    </Suspense>
  );
});

export default Icon;
