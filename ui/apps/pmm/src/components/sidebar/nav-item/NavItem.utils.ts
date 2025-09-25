import { NavItem } from 'lib/types';
import { Link } from 'react-router-dom';

export const getLinkProps = (item: NavItem, url?: string) => {
  if (item.onClick) {
    return { onClick: item.onClick };
  }

  if (item.target && item.url) {
    return {
      component: 'a',
      target: item.target,
      href: url,
    };
  }

  return {
    to: url,
    relative: false,
    component: Link,
  };
};

export const shouldShowBadge = (item: NavItem, expanded: boolean): boolean => {
  if (item.badge && !expanded) {
    return true;
  }

  return (
    item.children?.some((child) => shouldShowBadge(child, expanded)) || false
  );
};
