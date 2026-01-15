import { FunctionComponent, lazy, LazyExoticComponent, SVGProps } from 'react';
import { IconName } from './Icon.types';
import { DYNAMIC_ICON_IMPORT_MAP } from './Icon.constants';

const IconCache: Partial<
  Record<
    IconName,
    LazyExoticComponent<FunctionComponent<SVGProps<SVGSVGElement>>>
  >
> = {};

export const loadIcon = (name: IconName) => {
  if (!DYNAMIC_ICON_IMPORT_MAP[name]) {
    return null;
  }

  if (!IconCache[name]) {
    IconCache[name] = lazy(DYNAMIC_ICON_IMPORT_MAP[name]);
  }

  return IconCache[name];
};
