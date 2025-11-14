import { matchPath } from 'react-router-dom';
import { NavItem } from 'types/navigation.types';

export const findActiveNavItem = (
  navtree: NavItem[] | NavItem,
  pathname: string
): NavItem | undefined => {
  const roots = Array.isArray(navtree) ? navtree : [navtree];

  let active: { item: NavItem; depth: number } | undefined;

  const findActive = (item: NavItem, depth: number) => {
    if (item.children) {
      for (const child of item.children) {
        findActive(child, depth + 1);
      }
    }

    if (isActive(item, pathname)) {
      if (!active || depth > active.depth) {
        active = { item, depth };
      }
    }
  };

  for (const root of roots) {
    findActive(root, 0);
  }

  return active?.item;
};

export const isActive = (item: NavItem, pathname: string): boolean => {
  if (item.isDivider || !item.url) {
    return false;
  }

  const exactMatch = matchesUrl(pathname, item.url);
  const additionalMatch = item?.matches?.some((match) =>
    matchesUrl(pathname, item.url!, match)
  );

  return Boolean(exactMatch || additionalMatch);
};

const matchesUrl = (pathname: string, url: string, match?: string) => {
  const path = normalizePath(url);

  if (!match) {
    return !!matchPath({ path, end: true }, pathname);
  }

  if (match === '*') {
    return !!matchPath({ path: path + '/*', end: true }, pathname);
  }

  return !!matchPath(
    {
      path: normalizePath(match),
      end: true,
    },
    pathname
  );
};

const normalizePath = (path: string): string => {
  const withSlash = path.startsWith('/') ? path : `/${path}`;
  return withSlash.replace(/\/{2,}/g, '/');
};
