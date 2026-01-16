import { Icon } from 'components/icon';
import { NavItem } from 'types/navigation.types';
import { ComponentType, FC } from 'react';

interface Props {
  icon: NonNullable<NavItem['icon']>;
}

const NavItemIcon: FC<Props> = ({ icon }) => {
  if (typeof icon === 'string') {
    return <Icon name={icon} />;
  }

  // support also memoized components
  if (typeof icon === 'function' || typeof icon === 'object') {
    const NavIcon = icon as ComponentType;
    return <NavIcon />;
  }

  // fallback
  return null;
};

export default NavItemIcon;
