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
  if (item.type === 'menu-text' || item.type === 'menu-section' || !item.url) {
    return false;
  }

  // By path alone: a link can carry query params, such as PMM HA's namespace.
  const [url] = item.url.split('?');
  const exactMatch = matchesUrl(pathname, url);
  const additionalMatch = item?.matches?.some((match) =>
    matchesUrl(pathname, url, match)
  );

  return Boolean(exactMatch || additionalMatch);
};

// Grafana swaps a dashboard link's slug, or adds a missing one, for one made
// from the dashboard's title, so a dashboard matches by its UID alone.
const DASHBOARD_PATH = /^(.*\/d\/[^/]+)(?:\/[^/]*)?$/;

const matchesUrl = (pathname: string, url: string, match?: string) => {
  const path = normalizePath(url);

  if (!match) {
    const dashboard = path.match(DASHBOARD_PATH)?.[1];

    return !!matchPath(
      { path: dashboard ? `${dashboard}/*` : path, end: true },
      pathname
    );
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
