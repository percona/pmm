import { Icon } from 'components/icon';
import { NavItem } from 'lib/types';
import { FC } from 'react';

interface Props {
  icon: NonNullable<NavItem['icon']>;
}

const NavItemIcon: FC<Props> = ({ icon: NavIcon }) =>
  typeof NavIcon === 'string' ? <Icon name={NavIcon} /> : <NavIcon />;

export default NavItemIcon;
