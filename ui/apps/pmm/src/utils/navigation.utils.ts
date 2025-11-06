import { NavItem } from 'lib/types';
import { matchPath } from 'react-router-dom';

export const findActiveNavItem = (
  navtree: NavItem[] | NavItem,
  pathname: string
): NavItem | undefined => {
  for (const navItem of Array.isArray(navtree) ? navtree : [navtree]) {
    const activeItem = findNavItem(navItem, (item) => isActive(item, pathname));

    if (activeItem) {
      return activeItem;
    }
  }

  return undefined;
};

const findNavItem = (
  item: NavItem,
  predicate: (item: NavItem) => boolean
): NavItem | undefined => {
  if (predicate(item)) {
    return item;
  }

  for (const child of item.children || []) {
    const match = findNavItem(child, predicate);

    if (match) {
      return match;
    }
  }

  return undefined;
};

export const isActive = (item: NavItem, pathname: string): boolean => {
  if (item.isDivider || !item.url) {
    return false;
  }

  const exactMatch = matchesUrl(pathname, item.url);
  const additionalMatch = item?.matches?.some((match) =>
    matchesUrl(pathname, item.url!, match)
  );

  // check if first child is active and prefer it to be active item
  if (item.children?.length && isActive(item.children[0], pathname)) {
    return false;
  }

  return Boolean(exactMatch || additionalMatch);
};

const matchesUrl = (pathname: string, url: string, match?: string) => {
  if (!match) {
    return !!matchPath(
      {
        path: url,
        end: true,
      },
      pathname
    );
  } else if (match === '*') {
    return !!matchPath(
      {
        path: url + '/*',
        end: true,
      },
      pathname
    );
  } else {
    return !!matchPath(
      {
        path: match,
        end: true,
      },
      pathname
    );
  }
};
