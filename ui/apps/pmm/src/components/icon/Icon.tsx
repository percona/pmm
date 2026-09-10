import { FC, memo, Suspense } from 'react';
import { IconProps } from './Icon.types';
import { DYNAMIC_ICON_IMPORT_MAP, VIEWBOX_MAP } from './Icon.constants';
import SvgIcon from '@mui/material/SvgIcon';
import Box from '@mui/material/Box';
import { loadIcon } from './Icon.utils';
import { IconBoundary } from './Icon.boundary';

const Icon: FC<IconProps> = memo(({ name, ...props }) => {
  if (!DYNAMIC_ICON_IMPORT_MAP[name]) {
    return null;
  }

  const Icon = loadIcon(name);
  const placeholder = (
    <Box
      sx={{
        width: props.width || 24,
        height: props.height || 24,
        ...props.sx,
      }}
    />
  );

  // The boundary sits above the Suspense so it catches the import failing, not just
  // the wait for it.
  return (
    <IconBoundary fallback={placeholder}>
      <Suspense fallback={placeholder}>
        <SvgIcon component={Icon} viewBox={VIEWBOX_MAP[name]} {...props} />
      </Suspense>
    </IconBoundary>
  );
});

export default Icon;
