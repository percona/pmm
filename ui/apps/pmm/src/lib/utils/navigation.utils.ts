import { NavItem } from 'lib/types';
import { matchPath } from 'react-router-dom';

export const isActive = (item: NavItem, pathname: string): boolean => {
  if (item.children?.length) {
    return item.children.some((child) => isActive(child, pathname));
  }

  return !!matchPath(
    {
      path: item.url,
      end: false,
    },
    pathname
  );
};
