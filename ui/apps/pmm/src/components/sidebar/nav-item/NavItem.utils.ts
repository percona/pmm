import { NavItem } from 'lib/types';
import { Link } from 'react-router-dom';

export const getLinkProps = (item: NavItem, url?: string) => {
  if (item.onClick) {
    return { onClick: item.onClick };
  }

  if (item.target && item.url) {
    return {
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
