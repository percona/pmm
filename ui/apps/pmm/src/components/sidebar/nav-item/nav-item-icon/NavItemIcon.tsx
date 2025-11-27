import { Icon } from 'components/icon';
import { NavItem } from 'types/navigation.types';
import { FC } from 'react';

interface Props {
  icon: NonNullable<NavItem['icon']>;
}

const NavItemIcon: FC<Props> = ({ icon: NavIcon }) => {
  if (typeof NavIcon === 'string') {
    return <Icon name={NavIcon} />;
  }

  if (typeof NavIcon === 'function') {
    return <NavIcon />;
  }

  return NavIcon;
};

export default NavItemIcon;
