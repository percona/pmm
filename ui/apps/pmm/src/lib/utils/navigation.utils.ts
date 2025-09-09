import { NavItem } from 'lib/types';
import { matchPath } from 'react-router-dom';

export const isActive = (item: NavItem, pathname: string): boolean => {
  if (item.isDivider || !item.url) {
    return false;
  }

  const matches = !!matchPath(
    {
      path: item.url,
      end: true,
    },
    pathname
  );

  return matches || !!item.children?.some((child) => isActive(child, pathname));
};
