import { Icon } from 'components/icon';
import { NavItem } from 'types/navigation.types';
import { ComponentType, FC, isValidElement } from 'react';

interface Props {
  icon: NonNullable<NavItem['icon']>;
}

const NavItemIcon: FC<Props> = ({ icon }) => {
  if (typeof icon === 'string') {
    return <Icon name={icon} />;
  }

  // elements (such as <NavItemIcon icon={<TestIcon />} />)
  if (isValidElement(icon)) {
    return icon;
  }

  // support also memoized components  (such as <NavItemIcon icon={TestIcon} />)
  if (typeof icon === 'function' || typeof icon === 'object') {
    const NavIcon = icon as ComponentType;
    return <NavIcon />;
  }

  // fallback
  return null;
};

export default NavItemIcon;
