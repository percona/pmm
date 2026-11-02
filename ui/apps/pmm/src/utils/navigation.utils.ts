import { matchPath } from 'react-router-dom';
import { NavItem } from 'types/navigation.types';

export const findActiveNavItem = (
  navtree: NavItem[] | NavItem,
  pathname: string,
  search = ''
): NavItem | undefined => {
  const roots = Array.isArray(navtree) ? navtree : [navtree];

  let active: { item: NavItem; depth: number } | undefined;

  const findActive = (item: NavItem, depth: number) => {
    if (item.children) {
      for (const child of item.children) {
        findActive(child, depth + 1);
      }
    }

    if (isActive(item, pathname, search)) {
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

export const isActive = (
  item: NavItem,
  pathname: string,
  search = ''
): boolean => {
  if (
    item.type === 'menu-divider' ||
    item.type === 'menu-text' ||
    item.type === 'menu-section' ||
    !item.url
  ) {
    return false;
  }

  const [url, query] = item.url.split('?');

  // Two entries can share a page and differ only in the query string (a pinned
  // filtered view next to the plain page), so a url with a query string claims
  // the page only when the location carries every one of its params. The
  // `matches` patterns stay path-only.
  const exactMatch =
    matchesUrl(pathname, url) && (!query || hasParams(search, query));
  const additionalMatch = item?.matches?.some((match) =>
    matchesUrl(pathname, url, match)
  );

  return Boolean(exactMatch || additionalMatch);
};

const hasParams = (search: string, query: string): boolean => {
  const current = new URLSearchParams(search);

  return Array.from(new URLSearchParams(query)).every(
    ([key, value]) => current.get(key) === value
  );
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
